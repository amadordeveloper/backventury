package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type movie struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Genre       []string `json:"genre"`
	RelasedYear string   `json:"relasedYear"`
	Rating      float64  `json:"rating"`
}

func getMovieById(c *gin.Context) {
	id := c.Params.ByName("id")
	movieGet := selectMovie(id)
	if movieGet.ID != "" {
		c.IndentedJSON(http.StatusOK, movieGet)
		return
	}

	stringSearch := "i=" + id
	movieImdb := callImdb(stringSearch)
	if movieImdb.ID != "" {
		c.IndentedJSON(http.StatusOK, movieImdb)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
	}

}

func getMovieByYear(c *gin.Context) {
	year := c.Params.ByName("year")

	movieGet := selectMovies(year, "year")

	if len(movieGet) > 0 {
		c.IndentedJSON(http.StatusOK, movieGet)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Movies not found"})
}

func getMovieByGenre(c *gin.Context) {
	genre := c.Params.ByName("genre")

	movieGet := selectMovies(genre, "genre")

	if len(movieGet) > 0 {
		c.IndentedJSON(http.StatusOK, movieGet)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
}

func getMovieByRating(c *gin.Context) {
	rating := c.Params.ByName("rating")
	condition := c.Params.ByName("condition")

	movieGet := selectMovies(rating, condition)

	if len(movieGet) > 0 {
		c.IndentedJSON(http.StatusOK, movieGet)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
}

/* Against GOIMDB */

func callImdb(search string) movie {
	APIkey := "f8714b90"
	url := "http://www.omdbapi.com/?apikey=" + APIkey + "&" + search
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	//return string(body)

	var decode map[string]interface{}

	json.Unmarshal(body, &decode)

	if decode["Response"] == "False" {
		return movie{}
	} else {

		genresArr := strings.Split(decode["Genre"].(string), ",")
		for i, g := range genresArr {
			genresArr[i] = strings.TrimSpace(g)
		}

		rate, _ := strconv.ParseFloat(decode["imdbRating"].(string), 64)
		var movie = movie{
			ID:          decode["imdbID"].(string),
			Title:       decode["Title"].(string),
			Genre:       genresArr,
			RelasedYear: decode["Year"].(string),
			Rating:      rate,
		}
		saveMovie(movie)
		return movie
	}

}

/* Against Firebase */

func saveMovie(movie movie) bool {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "backventury", option.WithCredentialsFile("backventury-firebase-adminsdk-duuey-9c8ae8a2fe.json"))
	if err != nil {
		panic(err)
	}

	defer client.Close()

	_, _, err = client.Collection("movies").Add(ctx, map[string]interface{}{
		"id":          movie.ID,
		"title":       movie.Title,
		"genre":       movie.Genre,
		"relasedYear": movie.RelasedYear,
		"rating":      movie.Rating,
	})

	if err != nil {
		panic(err)
	}
	return true
}

func selectMovie(keyword string) movie {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "backventury", option.WithCredentialsFile("backventury-firebase-adminsdk-duuey-9c8ae8a2fe.json"))
	if err != nil {
		panic(err)
	}

	defer client.Close()
	query := client.Collection("movies").Where("id", "==", keyword)

	iter := query.Documents(ctx)
	var movieData movie
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
		}

		movieData = movie{
			ID:          doc.Data()["id"].(string),
			Title:       doc.Data()["title"].(string),
			Genre:       doc.Data()["genre"].([]string),
			RelasedYear: doc.Data()["relasedYear"].(string),
			Rating:      doc.Data()["rating"].(float64),
		}
	}
	fmt.Println(movieData.Title)
	return movieData

}

func selectMovies(keyword string, typeSearch string) []movie {
	fmt.Println(keyword)
	fmt.Println(typeSearch)
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "backventury", option.WithCredentialsFile("backventury-firebase-adminsdk-duuey-9c8ae8a2fe.json"))
	if err != nil {
		panic(err)
	}

	defer client.Close()

	var iter *firestore.DocumentIterator

	switch typeSearch {
	case "year":
		yearArr := strings.Split(keyword, "-")
		fmt.Println(yearArr)
		if len(yearArr) == 1 {
			iter = client.Collection("movies").Where("relasedYear", "==", keyword).Documents(ctx)
		} else {
			iter = client.Collection("movies").Where("relasedYear", ">=", yearArr[0]).Where("relasedYear", "<=", yearArr[1]).Documents(ctx)
		}
	case "genre":
		genreArr := strings.Split(keyword, ",")

		iter = client.Collection("movies").Where("genre", "array-contains-any", genreArr).Documents(ctx)
	case "greater":
		rate, err := strconv.ParseFloat(keyword, 64)
		println(err)
		iter = client.Collection("movies").Where("rating", ">", rate).Documents(ctx)
	case "less":
		rate, _ := strconv.ParseFloat(keyword, 64)
		iter = client.Collection("movies").Where("rating", "<", rate).Documents(ctx)
	default:
		iter = client.Collection("movies").Documents(ctx)
	}
	fmt.Println(iter)
	var movieData []movie
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(doc.Data())
		genresInterface := doc.Data()["genre"].([]interface{})

		var genres []string
		for _, genre := range genresInterface {
			genres = append(genres, genre.(string))
		}
		println(doc.Data()["id"].(string))
		movieSingle := movie{
			ID:          doc.Data()["id"].(string),
			Title:       doc.Data()["title"].(string),
			Genre:       genres,
			RelasedYear: doc.Data()["relasedYear"].(string),
			Rating:      doc.Data()["rating"].(float64),
		}

		movieData = append(movieData, movieSingle)
	}
	fmt.Println(movieData)
	return movieData

}

func main() {
	router := gin.Default()
	//router.GET("/movies", getMovies)
	router.GET("/movieById/:id", getMovieById)
	router.GET("/movieByYear/:year", getMovieByYear)
	router.GET("/movieByGenre/:genre", getMovieByGenre)
	router.GET("/movieByRating/:rating/:condition", getMovieByRating)
	//router.GET("/imdb/:id", callImdb)
	router.Run(":8080")
}
