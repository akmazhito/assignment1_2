module github.com/akmazhito/assignment1_2

go 1.22

require (
    github.com/gin-gonic/gin v1.10.0
    github.com/google/uuid v1.6.0
    github.com/lib/pq v1.10.9
    github.com/rabbitmq/amqp091-go v1.10.0
    
    // 1. Меняем зависимость на имя КОРНЕВОГО модуля
    github.com/akmazhito/assignment1_2 v0.0.0 
    
    google.golang.org/grpc v1.64.0
    google.golang.org/protobuf v1.34.2
)

// 2. Указываем правильный локальный путь до корня (на 2 уровня вверх)
replace github.com/akmazhito/assignment1_2 => ../../