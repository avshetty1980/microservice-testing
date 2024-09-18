### Go microservice that handles users and policies, integrates with Kafka and DynamoDB, and provides full CRUD functionality. It includes unit testing using `testify` and integration testing through the use of interfaces.

### Structure

```
/cmd
  main.go
/internal
  /kafka
    producer.go
  /repository
    dynamo_repository.go
  /service
    user_service.go
  /handlers
    user_handler.go
  /models
    user.go
/tests
  /unit
    user_service_test.go
  /integration
    user_handler_integration_test.go
```

### Step 1: Models

```go
// internal/models/user.go
package models

type User struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Email    string   `json:"email"`
    Policies []string `json:"policies"`
}
```

### Step 2: DynamoDB Repository

```go
// internal/repository/dynamo_repository.go
package repository

import (
    "context"
    "errors"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
    "user-microservice/internal/models"
)

type UserRepository interface {
    CreateUser(ctx context.Context, user models.User) error
    GetUser(ctx context.Context, id string) (*models.User, error)
    UpdateUser(ctx context.Context, user models.User) error
    DeleteUser(ctx context.Context, id string) error
}

type DynamoUserRepository struct {
    db *dynamodb.DynamoDB
}

func NewDynamoUserRepository() *DynamoUserRepository {
    sess := session.Must(session.NewSession())
    db := dynamodb.New(sess)
    return &DynamoUserRepository{db: db}
}

func (r *DynamoUserRepository) CreateUser(ctx context.Context, user models.User) error {
    av, err := dynamodbattribute.MarshalMap(user)
    if err != nil {
        return err
    }
    input := &dynamodb.PutItemInput{
        Item:      av,
        TableName: aws.String("Users"),
    }
    _, err = r.db.PutItemWithContext(ctx, input)
    return err
}

func (r *DynamoUserRepository) GetUser(ctx context.Context, id string) (*models.User, error) {
    result, err := r.db.GetItemWithContext(ctx, &dynamodb.GetItemInput{
        TableName: aws.String("Users"),
        Key: map[string]*dynamodb.AttributeValue{
            "ID": {S: aws.String(id)},
        },
    })
    if err != nil {
        return nil, err
    }
    if result.Item == nil {
        return nil, errors.New("user not found")
    }

    var user models.User
    err = dynamodbattribute.UnmarshalMap(result.Item, &user)
    return &user, err
}

func (r *DynamoUserRepository) UpdateUser(ctx context.Context, user models.User) error {
    return r.CreateUser(ctx, user) // Simulate upsert
}

func (r *DynamoUserRepository) DeleteUser(ctx context.Context, id string) error {
    _, err := r.db.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
        TableName: aws.String("Users"),
        Key: map[string]*dynamodb.AttributeValue{
            "ID": {S: aws.String(id)},
        },
    })
    return err
}
```

### Step 3: Kafka Producer

```go
// internal/kafka/producer.go
package kafka

import (
    "github.com/segmentio/kafka-go"
    "log"
)

type KafkaProducer struct {
    Writer *kafka.Writer
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
    return &KafkaProducer{
        Writer: &kafka.Writer{
            Addr:     kafka.TCP(brokers...),
            Topic:    "user-events",
            Balancer: &kafka.LeastBytes{},
        },
    }
}

func (p *KafkaProducer) PublishEvent(key, value string) error {
    err := p.Writer.WriteMessages(nil, kafka.Message{
        Key:   []byte(key),
        Value: []byte(value),
    })
    if err != nil {
        log.Printf("Failed to publish message: %v", err)
    }
    return err
}
```

### Step 4: Service Layer

```go
// internal/service/user_service.go
package service

import (
    "context"
    "encoding/json"
    "user-microservice/internal/kafka"
    "user-microservice/internal/models"
    "user-microservice/internal/repository"
)

type UserService struct {
    repo    repository.UserRepository
    kafka   *kafka.KafkaProducer
}

func NewUserService(repo repository.UserRepository, kafka *kafka.KafkaProducer) *UserService {
    return &UserService{repo: repo, kafka: kafka}
}

func (s *UserService) CreateUser(ctx context.Context, user models.User) error {
    err := s.repo.CreateUser(ctx, user)
    if err == nil {
        s.publishEvent("create_user", user)
    }
    return err
}

func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
    return s.repo.GetUser(ctx, id)
}

func (s *UserService) UpdateUser(ctx context.Context, user models.User) error {
    err := s.repo.UpdateUser(ctx, user)
    if err == nil {
        s.publishEvent("update_user", user)
    }
    return err
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
    err := s.repo.DeleteUser(ctx, id)
    if err == nil {
        s.publishEvent("delete_user", models.User{ID: id})
    }
    return err
}

func (s *UserService) publishEvent(eventType string, user models.User) {
    msg, _ := json.Marshal(user)
    s.kafka.PublishEvent(eventType, string(msg))
}
```

### Step 5: Handlers

```go
// internal/handlers/user_handler.go
package handlers

import (
    "encoding/json"
    "net/http"
    "user-microservice/internal/models"
    "user-microservice/internal/service"
)

type UserHandler struct {
    service *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{service: svc}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var user models.User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := h.service.CreateUser(r.Context(), user); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    // logic to get a user by ID from URL params (omitted for brevity)
}
```

### Step 6: Unit Tests

```go
// tests/unit/user_service_test.go
package unit

import (
    "context"
    "testing"
    "user-microservice/internal/models"
    "user-microservice/internal/repository/mocks"
    "user-microservice/internal/service"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestCreateUser(t *testing.T) {
    mockRepo := new(mocks.UserRepository)
    mockKafka := new(mocks.KafkaProducer)
    svc := service.NewUserService(mockRepo, mockKafka)

    user := models.User{ID: "1", Name: "John"}

    mockRepo.On("CreateUser", mock.Anything, user).Return(nil)
    mockKafka.On("PublishEvent", "create_user", mock.Anything).Return(nil)

    err := svc.CreateUser(context.TODO(), user)

    assert.NoError(t, err)
    mockRepo.AssertCalled(t, "CreateUser", mock.Anything, user)
    mockKafka.AssertCalled(t, "PublishEvent", "create_user", mock.Anything)
}
```

### Step 7: Integration Tests

```go
// tests/integration/user_handler_integration_test.go
package integration

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "user-microservice/internal/handlers"
    "user-microservice/internal/models"
    "user-microservice/internal/repository"
    "user-microservice/internal/service"

    "github.com/stretchr/testify/assert"
)

func TestCreateUserIntegration(t *testing.T) {
    repo := repository.NewDynamoUserRepository()
    kafka := kafka.NewKafkaProducer([]string{"localhost:9092"})
    svc := service.NewUserService(repo, kafka)
    handler := handlers.NewUserHandler(svc)

    user := models.User{ID: "1", Name: "John", Email: "john@example.com"}
    userJSON, _ := json.Marshal(user)

    req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(userJSON))
    rec := httptest.NewRecorder()

    handler.CreateUser(rec, req)

    assert.Equal(t, http.StatusCreated, rec.Code)

    savedUser, _ := svc.GetUser(context.TODO(), "

1")
    assert.Equal(t, user.Name, savedUser.Name)
}
```
