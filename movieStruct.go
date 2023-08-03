package main

import pq "github.com/lib/pq"

type MovieDBResponseArray struct {
	Results []MovieDBResponse `json:"results"`
}

type MovieDBResponse struct {
	Title string `json:"title"`
	ID int `json:"id"`
	ReleaseYear string `json:"release_date"`
}

type MovieID struct {
	ID int `json:"id"`
}

type MovieDetails struct {
	Title string `json:"title"`
	Tagline string `json:"tagline"`
	Overview string `json:"overview"`
	Genres []Genre `json:"genres"`
	Revenue int `json:"revenue"`
	Poster string `json:"poster_path"`
	Actors []Actor `json:"actors"` 
	ReleaseYear string `json:"release_date"`
	Director string `json:"director"`
	Producer string `json:"distributor"`
	IMDB string `json:"imdb_id"`
	Collection MovieCollection `json:"belongs_to_collection"`
}

type selections struct {
	Date string `gorm:"column:date"`
	Movie string `gorm:"column:movie"`
	NumCorrect int `gorm:"column:num_correct"`
	NumIncorrect int `gorm:"column:num_incorrect"`
	Tagline string `gorm:"column:tagline"`
	Overview string `gorm:"column:overview"`
	Genres pq.StringArray `gorm:"type:text[]; column:genres"`
	Actors pq.StringArray `gorm:"type:text[]; column:actors"` 
	Revenue int `gorm:"column:revenue"`
	Poster string `gorm:"column:poster"`
	ReleaseYear string `gorm:"column:year"`
	Director string `gorm:"column:director"`
	Producer string `gorm:"column:producer"`
	IMDB string `gorm:"column:imdbId"`
	Collection string `gorm:"column:collection"`
}

type User struct {
	ID string `gorm:"column:id"`
	Username string `gorm:"column:username"`
	Password string `gorm:"column:password"`
	CreatedAt string `gorm:"column:created_at"`
	DeletedAt string `gorm:"column:deleted_at"`
	UpdatedAt string `gorm:"column:updated_at"`
	SolvedPuzzles int `gorm:"column:solved_puzzles"`
	FailedPuzzles int `gorm:"column:failed_puzzles"`
	LastSolvedPuzzle string `gorm:"column:last_solved_puzzle"`
	Active bool `gorm:"column:active"`
}

type GameStatus struct {
	Won bool `json:"won"`
}

type Users struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserInfo struct {
	UserID string `gorm:"column:id" json:"id"`
}

type Info struct {
	GuessedMovie MovieDetails `json:"GuessedMovie"`
	Compare Comparison `json:"Comparison"`
}

type Hint struct {
	Tagline string `json:"tagline"`
	Overview string `json:"overview"`
}

type MovieCollection struct {
	Name string `json:"name"`
}

type Comparison struct {
	Correct bool `json:"correct"`
	Collection bool `json:"collection"`
	YearComparison int `json:"yearComparison"`
	GrossComparison int `json:"revenueComparison"`
	DirectorComparison bool `json:"directorComparison"`
	Genres []Genre `json:"genres"`
	Actors []Actor `json:"actors"`
}

type Genre struct {
	GenreVal string `json:"name"`
}

type Actor struct {
	Name string `json:"name"`
	Headshot string `json:"profile_path"`
}

type Actors struct {
	Actors []Actor `json:"cast"`
}

type CrewMember struct {
	Name string `json:"name"`
	Job string `json:"job"`
}

type Crew struct {
	EntireCrew []CrewMember `json:"crew"`
}

type Producer struct {
	Name string `json:"name"`
}

type Producers struct {
	Companies []Producer `json:"production_companies"`
}

type Log struct {
	ID string `gorm:"column:id"`
	Severity string `gorm:"column:severity"`
	Content string `gorm:"column:content"`
	Timestamp string `gorm:"column:timestamp"`
}

type Input struct {
	Title string `json:"title"`
	ID int `json:"id"`
}

type Message struct {
	Text string `json:"text"`
}

type Response struct {
	Success bool `json:"success"`
	Context string `json:"context"`
}

type Validation struct {
	Correct bool `json:"correct"`
	Data UserInfo `json:"data"`
}

//lint:ignore U1000 Used for Swagger
type docs_ID struct {
	ID int `json:"id"`
}

//lint:ignore U1000 Used for Swagger
type docs_Title struct {
	Title string `json:"title"`
}