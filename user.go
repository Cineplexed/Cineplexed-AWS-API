package main

import (
	"time"
	"math/rand"
	"net/http"
	"io"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func makeUser(username string, password string) bool {
	var creds Users
	creds.Username = username
	creds.Password = password
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		log("ERROR", "Failed to encrypt user data")
		return false
	} else {
		var user = User{
			ID: uuid.New().String(), 
			Username: creds.Username, 
			Password: string(hash), 
			CreatedAt: time.Now().String(),
			DeletedAt: "",
			UpdatedAt: "",
			SolvedPuzzles: 0,
			FailedPuzzles: 0,
			LastSolvedPuzzle: "",
			Active: true}
		result := db.Create(&user)
		if result.Error != nil {
			log("ERROR", "Failed to post user data to postgres")
			return false
		} else {
			log("INFO", "Created user")
			return true
		}
	}
}

func validateUser(username string, password string) Validation {
	var validateCreds User
	db.Where("username = ?", username).First(&validateCreds)
	if validateCreds.Active {
		err := bcrypt.CompareHashAndPassword([]byte(validateCreds.Password), []byte(password))
		if err != nil {
			log("WARNING", "Incorrect credentials on log in")
			return Validation{Correct: false}
		} else {
			var key UserInfo
			db.Table("users").Where("username = ? AND password = ?", username, validateCreds.Password).First(&key)
			log("INFO", "Validated user")
			return Validation{Correct: true, Data: key}
		}
	} else {
		log("WARNING", "Attempted to sign into inactive account")
		return Validation{Correct: false}
	}
}

func updateUser(username string, password string, userId string) bool {
	var entry User
	result := db.Where("id = ?", userId).First(&entry)
	if entry.Active {
		if result.Error != nil {
			log("ERROR", "Invalid User-Id given")
			return false
		} else {
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				log("ERROR", "Failed to encrypt")
				return false
			} else {
				entry.Username = username
				entry.Password = string(hash)
				entry.UpdatedAt = time.Now().String()
				result := db.Save(&entry)
				if result.Error != nil {
					log("ERROR", "Could not post user info to postgres")
					return false
				} else {
					log("INFO", "Updated user info")
					return true
				}
			}
		}
	} else {
		log("WARNING", "Attempted to update inactive account")
		return false
	}
}

func deleteUser(userId string) bool {
	var entry User
	result := db.Where("id = ?", userId).First(&entry)
	if entry.Active {
		if result.Error != nil {
			log("ERROR", "Invalid User-Id given")
			return false
		} else {
			entry.Active = false
			entry.DeletedAt = time.Now().String()
			result := db.Save(&entry)
			if result.Error != nil {
				log("ERROR", "Failed to post deletion to postgres")
				return false
			} else {
				log("INFO", "Deleted user")
				return true
			}
		}
	} else {
		log("ERROR", "Attempted to delete an inactive account")
		return false
	}
}

func finishGame(status bool, userId string) bool {
	userFailed := false

	if len(userId) > 0 {
		var entry User
		result := db.Where("id = ?", userId).First(&entry)
		if entry.Active {
			if result.Error != nil {
				log("ERROR", "Invalid user ID entered")
				userFailed = true
				return false
			} else {
				if status {
					entry.SolvedPuzzles = entry.SolvedPuzzles + 1
					entry.LastSolvedPuzzle = time.Now().String()
				} else {
					entry.FailedPuzzles = entry.FailedPuzzles + 1
				}
				result := db.Save(&entry)
				if result.Error != nil {
					log("ERROR", "Failed to post info to postgres")
					userFailed = true
					return false
				} else {
					log("INFO", "Submitted daily win info")
				}
			}
		} else {
			log("ERROR", "Attempted to complete a puzzle on inactive account")
			userFailed = true
			return false
		}
	}
	if !userFailed {
		//User-Id not present
		var daily selections
		result := db.Last(&daily)
		if result.Error != nil {
			log("ERROR", "Failed to pull info from postgres")
			return false
		} else {
			//Update daily movie field to reflect win/loss
			if status {
				daily.NumCorrect = daily.NumCorrect + 1
			} else {
				daily.NumIncorrect = daily.NumIncorrect + 1
			}
			result := db.Model(&daily).Where("date = ?", daily.Date).Updates(map[string]interface{} {
				"NumCorrect": daily.NumCorrect,
				"NumIncorrect": daily.NumIncorrect, 
			})
			if result.Error != nil {
				log("ERROR", "Failed to post info to postgres")
				return false
			} else {
				log("INFO", "Submitted daily info")
				return true
			}
		}
	}
	return false
}

func getUnlimitedMovie(userId string) bool {
	var entry UserUnlimited
	result := db.Table("users").Where("id = ?", userId).First(&entry)
	if result.Error != nil {
		log("ERROR", "Failed to query postgres")
		return false
	} else {
		if entry.Active {
			page := rand.Intn(25) + 1
			req := randUrl + "?api_key=" + key + "&page=" + fmt.Sprint(page) 
			response, err := http.Get(req)
			if err != nil {
				log("ERROR", "Could not perform API call")
				return false
			} else {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					log("ERROR", "Could not read response")
					return false
				} else {
					var collection MovieDBResponseArray
					json.Unmarshal(body, &collection)
					index := rand.Intn(20)
					item := collection.Results[index]
					detailedEntry := getMovieWithDetail(item.ID)

					var arrGenres []string = make([]string, len(detailedEntry.GuessedMovie.Genres))
					for i := 0; i < len(detailedEntry.GuessedMovie.Genres); i++ {
						arrGenres[i] = string(detailedEntry.GuessedMovie.Genres[i].GenreVal)
					}

					var arrActors []string = make([]string, len(detailedEntry.GuessedMovie.Actors))
					for i := 0; i < len(detailedEntry.GuessedMovie.Actors); i++ {
						arrActors[i] = string(detailedEntry.GuessedMovie.Actors[i].Name)
					}

					entry.Movie = detailedEntry.GuessedMovie.Title
					entry.Tagline = detailedEntry.GuessedMovie.Tagline
					entry.Overview = detailedEntry.GuessedMovie.Overview
					entry.Genres = arrGenres
					entry.Actors = arrActors
					entry.Revenue = detailedEntry.GuessedMovie.Revenue
					entry.Poster = detailedEntry.GuessedMovie.Poster
					entry.ReleaseYear = detailedEntry.GuessedMovie.ReleaseYear
					entry.Director = detailedEntry.GuessedMovie.Director
					entry.Producer = detailedEntry.GuessedMovie.Producer
					entry.IMDB = detailedEntry.GuessedMovie.IMDB
					entry.Collection = detailedEntry.GuessedMovie.Collection.Name

					result := db.Table("users").Save(&entry)
					if result.Error != nil {
						log("ERROR", "Could not save unlimited movie")
						return false
					} else {
						log("INFO", "Updated unlimited movie")
						return true
					}
				}
			}
		} else {
			log("WARNING", "Attempted to create unlimited movie on inactive account")
			return false
		}
	}
}

func solvedUnlimited(userId string) bool {
	var entry UserUnlimited
	db.Table("users").Where("id = ?", userId).First(&entry)
	if entry.Active {
		entry.UnlimitedSolves = entry.UnlimitedSolves + 1
		result := db.Table("users").Save(&entry)
		if result.Error != nil {
			log("ERROR", "Could not store player unlimited win in postgres")
			return false 
		} else {
			log("INFO", "Stored player win")
			return true
		}
	} else {
		log("WARNING", "Attempted to log win on inactive account")
		return false
	}
}