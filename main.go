package filmoteka

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Actor struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Gender      string    `json:"gender"`
	DateOfBirth time.Time `json:"date_of_birth"`
	Movies      []Movie
}

type Movie struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ReleaseDate time.Time `json:"release_date"`
	Rating      int64     `json:"rating"`
	Actors      []Actor   `json:"actors"`
}

func connectDB() (*sql.DB, error) {
	connStr := "user=postgres password=mysecretpassword dbname=filmoteka sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}
func addActor(actor *Actor) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO actors(name, gender, date_of_birth) VALUES($1, $2, $3) RETURNING id")
	if err != nil {
		return err
	}
	defer stmt.Close()

	err = stmt.QueryRow(actor.Name, actor.Gender, actor.DateOfBirth).Scan(&actor.ID)
	if err != nil {
		return err
	}
	return nil
}

func updateActor(actor *Actor) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE actors SET name=$1, gender=$2, date_of_birth=$3 WHERE id=$4")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(actor.Name, actor.Gender, actor.DateOfBirth, actor.ID)
	return err
}

func deleteActor(id int64) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("DELETE FROM actors WHERE id=$1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	return err
}
func addMovie(movie *Movie) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO movies(name, description, release_date, rating) VALUES($1, $2, $3, $4) RETURNING id")
	if err != nil {
		return err
	}
	defer stmt.Close()

	err = stmt.QueryRow(movie.Name, movie.Description, movie.ReleaseDate, movie.Rating).Scan(&movie.ID)
	if err != nil {
		return err
	}

	for _, actor := range movie.Actors {
		stmt, err := db.Prepare("INSERT INTO movie_actors(movie_id, actor_id) VALUES($1, $2)")
		if err != nil {
			return err
		}defer stmt.Close()

		_, err = stmt.Exec(movie.ID, actor.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateMovie(movie *Movie) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE movies SET name=$1, description=$2, release_date=$3, rating=$4 WHERE id=$5")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(movie.Name, movie.Description, movie.ReleaseDate, movie.Rating, movie.ID)
	if err != nil {
		return err
	}

	stmt, err = db.Prepare("DELETE FROM movie_actors WHERE movie_id=$1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(movie.ID)
	if err != nil {
		return err
	}

	for _, actor := range movie.Actors {
		stmt, err := db.Prepare("INSERT INTO movie_actors(movie_id, actor_id) VALUES($1, $2)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.Exec(movie.ID, actor.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteMovie(id int64) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("DELETE FROM movies WHERE id=$1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	return err
}
func getMovies(sortBy string, sortOrder string, title string, actor string) ([]Movie, error) {
	db, err := connectDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var movies []Movie

	var stmt *sql.Stmt
	var args []interface{}

	if title != "" {
		stmt, args, err = buildQuery("name", title, sortBy, sortOrder)
	} else if actor != "" {
		stmt, args, err = buildQuery("actor_name", actor, sortBy, sortOrder)
	} else {
		stmt, args, err = buildQuery("", "", sortBy, sortOrder)
	}

	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var movie Movie
		err = rows.Scan(
			&movie.ID,
			&movie.Name,
			&movie.Description,
			&movie.ReleaseDate,
			&movie.Rating,
		)
		if err != nil {
			return nil, err
		}

		var actors []Actor
		stmt, err := db.Prepare("SELECT actor_id FROM movie_actors WHERE movie_id=$1")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		rows, err := stmt.Query(movie.ID)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var actorID int64
			err = rows.Scan(&actorID)
			if err != nil {
				return nil, err
			}

			var actor Actor
			stmt, err := db.Prepare("SELECT name, gender, date_of_birth FROM actors WHERE id=$1")
			if err != nil {
				return nil, err
			}
			defer stmt.Close()

			err = stmt.QueryRow(actorID).Scan(
				&actor.Name,
				&actor.Gender,
				&actor.DateOfBirth,
			)
			if err != nil {
				return nil, err
			}

			actors = append(actors, actor)
		}

		movie.Actors = actors
		movies = append(movies, movie)
	}

	return movies, nil
}

func getActors() ([]Actor, error) {
	db, err := connectDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var actors []Actor

	rows, err := db.Query("SELECT id, name, gender, date_of_birth FROM actors")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var actor Actor
		err = rows.Scan(
			&actor.ID,
			&actor.Name,
			&actor.Gender,
			&actor.DateOfBirth,
		)
		if err != nil {
			return nil, err
		}

		var movies []Movie
		stmt, err := db.Prepare("SELECT movie_id FROM movie_actors WHERE actor_id=$1")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		rows, err := stmt.Query(actor.ID)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var movieID int64
			err = rows.Scan(&movieID)
			if err != nil {
				return nil, err
			}

			var movie Movie
			stmt, err := db.Prepare("SELECT name, description, release_date, rating FROM movies WHERE id=$1")
			if err != nil {
				return nil, err
			}
			defer stmt.Close()

			err = stmt.QueryRow(movieID).Scan(
				&movie.Name,
				&movie.Description,
				&movie.ReleaseDate,
				&movie.Rating,
			)
			if err != nil {
				return nil, err
			}

			movies = append(movies, movie)
		}

		actor.Movies = movies
		actors = append(actors, actor)
	}

	return actors, nil
}
func buildQuery(column, value, sortBy, sortOrder string) (*sql.Stmt, []interface{}, error) {
	var stmt *sql.Stmt
	var args []interface{}

	where := ""
	if column != "" {
		where = "WHERE " + column + " ILIKE $" + strconv.Itoa(len(args) + 1)
		args = append(args, "%"+value+"%")
	}

	order := "ORDER BY name"
	if sortBy != "" {
		order = "ORDER BY " + sortBy
	}

	if sortOrder == "desc" {
		order += " DESC"
	}

	query := "SELECT m.id, m.name, m.description, m.release_date, m.rating, a.id AS actor_id, a.name AS actor_name, a.gender, a.date_of_birth FROM movies m " +
		"JOIN movie_actors ma ON m.id = ma.movie_id " +
		"JOIN actors a ON ma.actor_id = a.id " + where + " " + order

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, nil, err
	}

	return stmt, args, nil
}
func createActorEndpoint(w http.ResponseWriter, r *http.Request) {
	var actor Actor
	err := json.NewDecoder(r.Body).Decode(&actor)if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = addActor(&actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actor)
}

func updateActorEndpoint(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var actor Actor
	err := json.NewDecoder(r.Body).Decode(&actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actor.ID, _ = strconv.ParseInt(id, 10, 64)

	err = updateActor(&actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actor)
}

func deleteActorEndpoint(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	actorID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = deleteActor(actorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func createMovieEndpoint(w http.ResponseWriter, r *http.Request) {
	var movie Movie
	err := json.NewDecoder(r.Body).Decode(&movie)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = addMovie(&movie)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movie)
}

func updateMovieEndpoint(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var movie Movie
	err := json.NewDecoder(r.Body).Decode(&movie)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	movie.ID, _ = strconv.ParseInt(id, 10, 64)

	err = updateMovie(&movie)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movie)
}

func deleteMovieEndpoint(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	movieID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = deleteMovie(movieID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getMoviesEndpoint(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")
	title := r.URL.Query().Get("title")
	actor := r.URL.Query().Get("actor")

	movies, err := getMovies(sortBy, sortOrder, title, actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

func getActorsEndpoint(w http.ResponseWriter, r *http.Request) {
	actors, err := getActors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actors)
}

func searchMoviesEndpoint(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	actor := r.URL.Query().Get("actor")

	movies, err := searchMovies(title, actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

func searchActorsEndpoint(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	actors, err := searchActors(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actors)
}