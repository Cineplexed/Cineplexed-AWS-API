package main

import (
    "encoding/json"
    "fmt"
    "strconv"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "os"
    "time"
    "math/rand"

    "github.com/google/uuid"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
)

var db *gorm.DB = connectDB()

func HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    var ApiResponse events.APIGatewayProxyResponse

    fmt.Println("Key Param: " + request.QueryStringParameters["key"])
    if len(request.QueryStringParameters["key"]) != 0 {
        var key Key
        result := db.Table("keys").Where("api_key = ?", request.QueryStringParameters["key"]).First(&key)
        if result.Error != nil {
            log("ERROR", "User entered an invalid API key")
            ApiResponse = events.APIGatewayProxyResponse{Body: "Please enter a valid API key"}
        } else {

            key.NumCalls = key.NumCalls + 1
            db.Table("keys").Where("api_key = ?", request.QueryStringParameters["key"]).Save(&key)

            endpoint := request.PathParameters["proxy"]
            switch endpoint {
            case "getMovieOptions":
                if request.HTTPMethod == "GET" && key.Scope >= 1 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "getMovieDetails":
                if request.HTTPMethod == "GET" && key.Scope >= 1 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "getHint":
                if request.HTTPMethod == "GET" && key.Scope >= 1 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "makeUser": 
                if request.HTTPMethod == "POST" && key.Scope >= 2 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "validateUser": 
                if request.HTTPMethod == "POST" && key.Scope >= 2 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "updateUser":
                if request.HTTPMethod == "POST" && key.Scope >= 2 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "deleteUser":
                if request.HTTPMethod == "DELETE" && key.Scope >= 2{
                    if deleteUser(request.Headers["User-Id"]) {
                        ApiResponse = events.APIGatewayProxyResponse{Body: "Deleted user", StatusCode: 200}
                    } else {
                        ApiResponse = events.APIGatewayProxyResponse{Body: "Something went wrong", StatusCode: 500}
                    }
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "finishGame":
                if request.HTTPMethod == "POST" && key.Scope >= 1 {
                    var entry GameStatus
                    json.Unmarshal([]byte(request.Body), &entry)
                    if finishGame(entry.Won, request.Headers["User-Id"]) {
                        ApiResponse = events.APIGatewayProxyResponse{Body: "Submitted info", StatusCode: 200}
                    } else {
                        ApiResponse = events.APIGatewayProxyResponse{Body: "Something went wrong", StatusCode: 500}
                    }
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "getUnlimited":
                if request.HTTPMethod == "GET" && key.Scope >= 2 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "getUnlimitedMovieDetails":
                if request.HTTPMethod == "GET" && key.Scope >= 2 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "finishUnlimited":
                if request.HTTPMethod == "POST" && key.Scope >= 2 {
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
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            case "makeKey":
                if request.HTTPMethod == "POST" && key.Scope == 3 {
                    var key Key
                    key.Manager = request.QueryStringParameters["manager"]
                    key.Scope, _ = strconv.Atoi(request.QueryStringParameters["scope"])
                    if key.Manager == "" || key.Scope == 0 {
                        log("ERROR", "manager missing for key creation")
                        ApiResponse = events.APIGatewayProxyResponse{Body: "Error: missing query parameter manager or scope", StatusCode: 500}
                    } else {
                        key.APIKey = generateRandomString(20)
                        key.CreatedAt = time.Now().String()
                        key.NumCalls = 0

                        result := db.Table("keys").Create(&key)
                        if result.Error != nil {
                            log("ERROR", "Failed to post data to postgres")
                            ApiResponse = events.APIGatewayProxyResponse{Body: "Failed to create key in postgres", StatusCode: 500}
                        } else {
                            fmt.Println("Created key")
                            bytes, _ := json.Marshal(&key)
                            ApiResponse = events.APIGatewayProxyResponse{Body: string(bytes), StatusCode: 200}
                        }
                    }
                } else {
                    log("WARNING", "Wrong endpoint or bad key scope")
                    ApiResponse = events.APIGatewayProxyResponse{Body: "Please switch to the correct endpoint or upgrade to a key of higher scope", StatusCode: 500}
                }
            default:
                ApiResponse = events.APIGatewayProxyResponse{Body: "Please enter a valid endpoint", StatusCode: 500}
            }
        }
    } else {
        fmt.Println("No KEY PRESENT")
        log("ERROR", "User made call without API Key present")
        ApiResponse = events.APIGatewayProxyResponse{Body: "Please enter a valid API key", StatusCode: 500}
    }

    return ApiResponse, nil
}

func generateRandomString(length int) string {

    const letterAndNumber = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = letterAndNumber[rand.Intn(len(letterAndNumber))]
	}
	return string(result)
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