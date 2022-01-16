package main

import (
	"github.com/StalkR/imdb"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
)
import "github.com/gin-gonic/gin"
import _ "github.com/mattn/go-sqlite3"

type Movie struct {
	gorm.Model
	Title       string
	ReleaseYear int
	Rating      float64
	Genres      string
}

type UpdateParams struct {
	Rating float64 `json:"Rating"`
	Genres string  `json:"Genres"`
}

func main() {
	client := http.DefaultClient
	router := gin.Default()

	db, err := gorm.Open(sqlite.Open("./identifier.sqlite"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Movie{})

	// Find a title by partial name or retrieve it from IMDB
	router.GET("/movies/:name", func(c *gin.Context) {
		name := c.Param("name")
		var movie Movie

		// Attempt to find in local database
		err := db.First(&movie, "title = ?", name).Error

		if err != nil {
			// Otherwise, get partial matches from IMDB API
			movies, err := imdb.SearchTitle(client, name)

			if err != nil {
				c.AbortWithStatus(http.StatusNotFound)
			} else {

				// Finally, get full movie info from IMDB and store in our local database.
				foundMovie, _ := imdb.NewTitle(client, movies[0].ID)

				rating, _ := strconv.ParseFloat(foundMovie.Rating, 64)
				newMovie := Movie{
					Title:       foundMovie.Name,
					ReleaseYear: foundMovie.Year,
					Rating:      rating,
					Genres:      strings.Join(foundMovie.Genres, ","),
				}
				db.Create(&newMovie)

				c.JSON(http.StatusOK, newMovie)
			}

		} else {
			c.JSON(http.StatusOK, movie)
		}

	})

	// Updates only rating and genres by using our database internal movie ID
	router.PATCH("/movies/update/:id", func(c *gin.Context) {
		movieId := c.Param("id")
		var body UpdateParams
		err := c.ShouldBindBodyWith(&body, binding.JSON)

		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		} else {
			var movie Movie
			db.First(&movie, movieId)

			movie.Rating = body.Rating
			movie.Genres = body.Genres
			db.Save(&movie)

			c.JSON(http.StatusAccepted, movie)
		}

	})

	// Search movie by using its ID
	router.GET("/movies/by-id/:id", func(c *gin.Context) {
		movieId := c.Param("id")
		var movie Movie

		err := db.First(&movie, movieId).Error

		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.JSON(http.StatusOK, movie)
		}

	})

	// Retrieve a movie list by using its release year
	router.GET("/movies/by-year/:year", func(c *gin.Context) {
		year := c.Param("year")
		var movies []Movie
		db.Where("release_year = ?", year).Find(&movies)
		c.JSON(http.StatusOK, movies)

	})

	// Retrieve a movie list by using a date range (using year)
	router.GET("/movies/by-range/:start-year/:end-year", func(c *gin.Context) {
		startYear := c.Param("start-year")
		endYear := c.Param("end-year")
		var movies []Movie
		db.Where("release_year BETWEEN ? AND ?", startYear, endYear).Find(&movies)
		c.JSON(http.StatusOK, movies)
	})

	// Retrieve a movie list with higher rating than the requested value
	router.GET("/movies/by-higher-rating/:rating", func(c *gin.Context) {
		rating := c.Param("rating")
		var movies []Movie
		db.Where("rating > ?", rating).Find(&movies)
		c.JSON(http.StatusOK, movies)
	})

	// Retrieve a movie list with lower rating than the requested value
	router.GET("/movies/by-lower-rating/:rating", func(c *gin.Context) {
		rating := c.Param("rating")
		var movies []Movie
		db.Where("rating < ?", rating).Find(&movies)
		c.JSON(http.StatusOK, movies)
	})

	// Retrieve a movie list with the specified genre
	router.GET("/movies/by-genre/:genre", func(c *gin.Context) {
		genre := c.Param("genre")
		var movies []Movie
		db.Where("genres LIKE ?", "%"+genre+"%").Find(&movies)
		c.JSON(http.StatusOK, movies)
	})

	// Run the server and wait for request in 8080 port
	router.Run(":8080")
}
