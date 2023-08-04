package main

import (
    "encoding/json"
    "fmt"
    "strconv"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "os"
    "time"

    "github.com/google/uuid"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
)

var db *gorm.DB = connectDB()

func HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    var ApiResponse events.APIGatewayProxyResponse

    endpoint := request.PathParameters["proxy"]

    switch endpoint {
    case "getMovieOptions":
        if request.HTTPMethod == "GET" {
            name := request.QueryStringParameters["name"]
            if name != "" {
                response := getMovieByName(name)
                bytes, _ := json.Marshal(response)
                ApiResponse = events.APIGatewayProxyResponse{Body: string(bytes), StatusCode: 200}
                log("INFO", "Got movie options")
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Query Parameter name missing", StatusCode: 500}
                log("WARNING", "Query parameter missing when getting movie options")
            }
        }
    case "getMovieDetails":
        if request.HTTPMethod == "GET" {
            checkTime()
            id := request.QueryStringParameters["id"]
            if id != "" {
                numId, _ := strconv.Atoi(id)
                response := getMovieWithDetail(numId)
                bytes, _ := json.Marshal(response)
                ApiResponse = events.APIGatewayProxyResponse{Body: string(bytes), StatusCode: 200}
                log("INFO", "Got movie details")
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Query Parameter id missing", StatusCode: 500}
                log("WARNING", "Query parameter missing when getting movie details")
            }
        }
    case "getHint":
        if request.HTTPMethod == "GET" {
            var entry selections
            result := db.Last(&entry)
            if result.Error != nil {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Request Failed", StatusCode: 500}
                log("ERROR", "Could not get hint")
            } else {
                var hint Hint
                hint.Tagline = entry.Tagline
                hint.Overview = entry.Overview
                bytes, _ := json.Marshal(hint)
                ApiResponse = events.APIGatewayProxyResponse{Body: string(bytes), StatusCode: 200}
                log("INFO", "Got hint")
            }
        }
    case "makeUser": 
        if request.HTTPMethod == "POST" {
            var entry Users
            json.Unmarshal([]byte(request.Body), &entry)
            if entry.Username == "" || entry.Password == "" {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Username or password is missing"}
            } else {
                if makeUser(entry.Username, entry.Password) {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Added User", StatusCode: 200}
                } else {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Something went wrong", StatusCode: 500}
                }
            }
        }
    case "validateUser": 
        if request.HTTPMethod == "POST" {
            var entry Users
            json.Unmarshal([]byte(request.Body), &entry)
            if entry.Username == "" || entry.Password == "" {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Username or password is missing"}
            } else {
                var data Validation = validateUser(entry.Username, entry.Password)
                if data.Correct {
                    bytes, _ := json.Marshal(data.Data)
                    ApiResponse = events.APIGatewayProxyResponse{Body: string(bytes), StatusCode: 200}
                } else {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Incorrect credentials", StatusCode: 500}
                }
            }
        }
    case "updateUser":
        if request.HTTPMethod == "POST" {
            var entry Users
            json.Unmarshal([]byte(request.Body), &entry)
            if entry.Username == "" || entry.Password == "" {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Username or password is missing", StatusCode: 500}
            } else {
                if updateUser(entry.Username, entry.Password, request.Headers["User-Id"]) {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Updated user", StatusCode: 200}
                } else {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Something went wrong", StatusCode: 500}
                }
            }
        }
    case "deleteUser":
        if request.HTTPMethod == "DELETE" {
            if deleteUser(request.Headers["User-Id"]) {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Deleted user", StatusCode: 200}
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Something went wrong", StatusCode: 500}
            }
        }
    case "finishGame":
        if request.HTTPMethod == "POST" {
            var entry GameStatus
            json.Unmarshal([]byte(request.Body), &entry)
            if finishGame(entry.Won, request.Headers["User-Id"]) {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Submitted info", StatusCode: 200}
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Something went wrong", StatusCode: 500}
            }
        }
    case "getUnlimited":
        if request.HTTPMethod == "GET" {
            id := request.Headers["User-Id"]
            if len(id) > 0 {
                if getUnlimitedMovie(id) {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Got new unlimited movie", StatusCode: 200}
                } else {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Failed to get new unlimited movie", StatusCode: 500}
                }
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: no user id found", StatusCode: 500}
            }
        }
    case "getUnlimitedMovieDetails":
        if request.HTTPMethod == "GET" {
            id := request.QueryStringParameters["id"]
            if id != "" {
                if request.Headers["User-Id"] != "" {
                    numId, _ := strconv.Atoi(id)
                    response := getUnlimitedMovieWithDetail(numId, request.Headers["User-Id"])
                    bytes, _ := json.Marshal(response)
                    ApiResponse = events.APIGatewayProxyResponse{Body: string(bytes), StatusCode: 200}
                    log("INFO", "Got movie details")
                } else {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Error: user id is missing", StatusCode: 500}
                    log("WARNING", "User ID missing when getting unlimited movie details")
                }
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Query Parameter id missing", StatusCode: 500}
                log("WARNING", "Query parameter missing when getting movie details")
            }
        }
    case "finishUnlimited":
        if request.HTTPMethod == "POST" {
            id := request.Headers["User-Id"]
            if id != "" {
                if (solvedUnlimited(id)) {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Submitted info", StatusCode: 200}
                } else {
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Failed to submit info", StatusCode: 500}
                }
            } else {
                ApiResponse = events.APIGatewayProxyResponse{Body: "Error: user id is missing", StatusCode: 500}
            }
        }
    default:
        ApiResponse = events.APIGatewayProxyResponse{Body: "Please enter a valid endpoint", StatusCode: 500}
    }
    return ApiResponse, nil
}

func log(severity string, content string) {
	var entry Log
	entry.ID = uuid.New().String()
	entry.Severity = severity
	entry.Content = content
    time.LoadLocation("America/New_York")
	entry.Timestamp = time.Now().String()
	result := db.Create(&entry)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
	}
}

func connectDB() *gorm.DB {
    dsn := os.Getenv("conString")
    fmt.Println(dsn)
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        fmt.Println(err.Error())
    } else {
        db.AutoMigrate(&selections{})
        fmt.Println("Connected to Database...")
        return db
    }
    return nil
}

func main() {
    getEnv()
    getTargetTime()
	lambda.Start(HandleRequest)
}