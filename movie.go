package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"io"
	"strings"
	"strconv"
	"math/rand"
	"time"
	"os"
)

var key string
var baseUrl string
var searchUrl string
var randUrl string

var tomorrow time.Time
var nextTime time.Time
var updatingDaily = false

func getEnv() {
	key = os.Getenv("movieDBKey")
	baseUrl = os.Getenv("movieDBRootUrl")
	searchUrl = os.Getenv("movieDBSearchUrl")
	randUrl = os.Getenv("movieDBRandUrl")
	log("INFO", "Got environment variables")
}

func getMovieByName(title string) MovieDBResponseArray {
	req := searchUrl + "?api_key=" + key + "&query=" + strings.ReplaceAll(strings.TrimSpace(title), " ", "+")
	response, err := http.Get(req)
	if err != nil {
		log("ERROR", "Could not get movies when getting movie options")
	} else {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log("ERROR", "could not read response when getting movie options")
		} else {
			var entry MovieDBResponseArray
			json.Unmarshal(body, &entry)

			var limitedEntry MovieDBResponseArray
			if (len(entry.Results) < 10) {
				limitedEntry.Results = make([]MovieDBResponse, len(entry.Results))
			} else {
				limitedEntry.Results = make([]MovieDBResponse, 10)
			}
			for i := 0; i < len(limitedEntry.Results); i++ {
				limitedEntry.Results[i] = entry.Results[i]
				if len(limitedEntry.Results[i].ReleaseYear) >= 4 {
					limitedEntry.Results[i].ReleaseYear = limitedEntry.Results[i].ReleaseYear[0:4]
				} else {
					limitedEntry.Results[i].ReleaseYear = "Unreleased"
				}
			}
			fmt.Println("INFO", "given options")
			return limitedEntry
		}
	}
	fmt.Println("ERROR", "Something went wrong with getting options")
	var entry MovieDBResponseArray
	return entry
}

func getMovieWithDetail(id int) Info {

	entry := getDetails(id)

	var daily selections
	db.Last(&daily)

	var compDetails Comparison
	compDetails.Correct = (daily.Movie == entry.Title)
	compDetails.Collection = ((daily.Collection != "" && entry.Collection.Name != "") && daily.Collection == entry.Collection.Name)
	dailyYear, _ := strconv.Atoi(daily.ReleaseYear)
	entryYear, _ := strconv.Atoi(entry.ReleaseYear)
	compDetails.YearComparison = (dailyYear - entryYear) * -1
	compDetails.GrossComparison = (daily.Revenue - entry.Revenue) * -1
	compDetails.DirectorComparison = (daily.Director == entry.Director)
	compDetails.ProducerComparison = (daily.Producer == entry.Producer)

	actorArr := make([]Actor, len(daily.Actors))
	for i := 0; i < len(actorArr); i++ {
		actorArr[i] = Actor{Name: daily.Actors[i]}
	}
	tempActorArr := getMatchingActors(Actors{Actors: actorArr}, Actors{Actors: entry.Actors})
	if len(tempActorArr) > 0 {
		compDetails.Actors = tempActorArr
	} else {
		compDetails.Actors = make([]Actor, 0)
	}

	genreArr := make([]Genre, len(daily.Genres))
	for i := 0; i < len(genreArr); i++ {
		genreArr[i] = Genre{GenreVal: daily.Genres[i]}
	}
	tempGenreArr := getMatchingGeres(genreArr, entry.Genres)
	if len(tempGenreArr) > 0 {
		compDetails.Genres = tempGenreArr
	} else {
		compDetails.Genres = make([]Genre, 0)
	}

	fmt.Println("INFO", "details given")

	var info Info
	info.GuessedMovie = entry
	info.Compare = compDetails

	return info
}

func getMatchingGeres(dailyArr []Genre, guessedArr []Genre) []Genre {
	count := 0
	index := 0
	for i := 0; i < len(dailyArr); i++ {
		for i2 := 0; i2 < len(guessedArr); i2++ {
			if dailyArr[i] == guessedArr[i2] {
				count++
			}
		}
	}
	finalArr := make([]Genre, count)
	for i := 0; i < len(dailyArr); i++ {
		for i2 := 0; i2 < len(guessedArr); i2++ {
			if dailyArr[i] == guessedArr[i2] {
				finalArr[index] = dailyArr[i]
				index++
			}
		}
	}
	return finalArr
}

func getMatchingActors(dailyActors Actors, guessActors Actors) []Actor {
	var finalArr []Actor
	count := 0
	index := 0
	for i := 0; i < len(dailyActors.Actors); i++ {
		for i2 := 0; i2 < len(guessActors.Actors); i2++ {
			if dailyActors.Actors[i].Name == guessActors.Actors[i2].Name {
				count++
			}
		}	
	}
	finalArr = make([]Actor, count)
	for i := 0; i < len(dailyActors.Actors); i++ {
		for i2 := 0; i2 < len(guessActors.Actors); i2++ {
			if dailyActors.Actors[i].Name == guessActors.Actors[i2].Name {
				finalArr[index] = Actor{Name: dailyActors.Actors[i].Name, Headshot: guessActors.Actors[i].Headshot}
				index++
			}
		}	
	}
	return finalArr
}

func getDailyMovie() {
	page := rand.Intn(25) + 1
	req := randUrl + "?api_key=" + key + "&page=" + fmt.Sprint(page) 
	response, err := http.Get(req)
	if err != nil {
		fmt.Println("ERROR API CALL")
	} else {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("ERROR READ RESPONSE")
		} else {
			var collection MovieDBResponseArray
			json.Unmarshal(body, &collection)
			index := rand.Intn(20)
			entry := collection.Results[index]
			detailedEntry := getMovieWithDetail(entry.ID)

			var arrGenres []string = make([]string, len(detailedEntry.GuessedMovie.Genres))
			for i := 0; i < len(detailedEntry.GuessedMovie.Genres); i++ {
				arrGenres[i] = string(detailedEntry.GuessedMovie.Genres[i].GenreVal)
			}

			var arrActors []string = make([]string, len(detailedEntry.GuessedMovie.Actors))
			for i := 0; i < len(detailedEntry.GuessedMovie.Actors); i++ {
				arrActors[i] = string(detailedEntry.GuessedMovie.Actors[i].Name)
			}

			var complete selections = selections{
				Date: nextTime.Format("2006") + "/" + nextTime.Format("01") + "/" + nextTime.Format("02"), 
				Movie: detailedEntry.GuessedMovie.Title, 
				NumCorrect: 0,
				NumIncorrect: 0,
				Tagline: detailedEntry.GuessedMovie.Tagline,
				Overview: detailedEntry.GuessedMovie.Overview,
				Genres: arrGenres,
				Actors: arrActors,
				Revenue: detailedEntry.GuessedMovie.Revenue,
				Poster: detailedEntry.GuessedMovie.Poster,
				ReleaseYear: detailedEntry.GuessedMovie.ReleaseYear,
				Director: detailedEntry.GuessedMovie.Director,
				Producer: detailedEntry.GuessedMovie.Producer,
				IMDB: detailedEntry.GuessedMovie.IMDB,
				Collection: detailedEntry.GuessedMovie.Collection.Name}
			if complete.Revenue == 0 {
				getDailyMovie()
			} else {
				db.Table("selections")
				result := db.Create(&complete)
				if result.Error != nil {
					log("ERROR", "Could not create new daily movie")
				} else {
					log("INFO", "Got new daily movie")
				}
			}
		}
	}
	getTargetTime()
	updatingDaily = false
}

func checkTime() {
	time.LoadLocation("America/New_York")
	fmt.Println("Checking Time: ")
	fmt.Println(time.Now())
	fmt.Println(nextTime)
	if time.Now().After(nextTime) && !updatingDaily {
		updatingDaily = true
		getDailyMovie()
	}
}

func getTargetTime() {
	var entry selections
	result := db.Last(&entry)
	if result.Error == nil {
		time.LoadLocation("America/New_York")
		fmt.Println(entry.Date)
		fmt.Println(len(entry.Date))
		if len(entry.Date) > 0 {
			tomorrow, _ = time.Parse("2006-01-02", strings.ReplaceAll(entry.Date, "/", "-"))
			nextTime = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day() + 1, 0, 0, 0, 0, time.Now().Location())
		} else {
			nextTime = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
		}
		log("INFO", "Got new target time")
	}
}

func getUnlimitedMovieWithDetail(id int, userId string) Info {
	var daily UserUnlimited
	db.Table("users").Where("id = ?", userId).First(&daily)

	entry := getDetails(id)

	var compDetails Comparison
	compDetails.Correct = (daily.Movie == entry.Title)
	compDetails.Collection = ((daily.Collection != "" && entry.Collection.Name != "") && daily.Collection == entry.Collection.Name)
	dailyYear, _ := strconv.Atoi(daily.ReleaseYear)
	entryYear, _ := strconv.Atoi(entry.ReleaseYear)
	compDetails.YearComparison = (dailyYear - entryYear) * -1
	compDetails.GrossComparison = (daily.Revenue - entry.Revenue) * -1
	compDetails.DirectorComparison = (daily.Director == entry.Director)
	compDetails.ProducerComparison = (daily.Producer == entry.Producer)

	actorArr := make([]Actor, len(daily.Actors))
	for i := 0; i < len(actorArr); i++ {
		actorArr[i] = Actor{Name: daily.Actors[i]}
	}
	tempActorArr := getMatchingActors(Actors{Actors: actorArr}, Actors{Actors: entry.Actors})
	if len(tempActorArr) > 0 {
		compDetails.Actors = tempActorArr
	} else {
		compDetails.Actors = make([]Actor, 0)
	}

	genreArr := make([]Genre, len(daily.Genres))
	for i := 0; i < len(genreArr); i++ {
		genreArr[i] = Genre{GenreVal: daily.Genres[i]}
	}
	tempGenreArr := getMatchingGeres(genreArr, entry.Genres)
	if len(tempGenreArr) > 0 {
		compDetails.Genres = tempGenreArr
	} else {
		compDetails.Genres = make([]Genre, 0)
	}

	var info Info
	info.GuessedMovie = entry
	info.Compare = compDetails
	return info
}

func getDetails(id int) MovieDetails {
	movieDetailReq := baseUrl + "/" + fmt.Sprint(id) + "?api_key=" + key
	response, err := http.Get(movieDetailReq)
	if err != nil {
		log("ERROR", "Could not get movie details when getting movie details")
	} else {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log("ERROR", "Could not read response when getting movie details")
		} else {
			var entry MovieDetails
			json.Unmarshal(body, &entry)
			
			var producers Producers
			json.Unmarshal(body, &producers)

			if len(producers.Companies) > 0 {
				entry.Producer = producers.Companies[0].Name
			} 
			if len(entry.ReleaseYear) >= 4 { 
				entry.ReleaseYear = entry.ReleaseYear[0:4]
			} else {
				entry.ReleaseYear = "Unreleased"
			}

			movieActorReq := baseUrl + "/" + fmt.Sprint(id) + "/credits?api_key=" + key
			response, err := http.Get(movieActorReq)
			if err != nil {
				log("ERROR", "Could not get cast details when getting movie details")
			} else {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					log("ERROR", "Could not read response when getting movie details")
				} else {
					var actors Actors
					json.Unmarshal(body, &actors)

					var crew Crew
					json.Unmarshal(body, &crew)

					for i := 0; i < len(crew.EntireCrew); i++ {
						if crew.EntireCrew[i].Job == "Director" {
							entry.Director = crew.EntireCrew[i].Name
							break
						}
					}

					var arr []Actor
					if len(actors.Actors) < 10 {
						arr = make([]Actor, len(actors.Actors))
					} else {
						arr = make([]Actor, 10)
					}
					for i := 0; i < len(actors.Actors) && i < len(arr); i++ {
						arr[i].Name = actors.Actors[i].Name
						arr[i].Headshot = actors.Actors[i].Headshot
					}
					entry.Actors = arr
					return entry
				}
			}
		}
	}
	return MovieDetails{}
}