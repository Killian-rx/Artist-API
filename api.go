package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type Itemm struct {
	Id           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

type Relation struct {
	Index []struct {
		Id             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

// Structure pour stocker les variables de la page
type PageVariables struct {
	FilteredArtists map[int]struct {
		Itemm
		DatesLocations map[string][]string
	}
	SearchTerm string
}

func main() {
	apiURL := "https://groupietrackers.herokuapp.com/api/artists"
	apiURL2 := "https://groupietrackers.herokuapp.com/api/relation"

	// Première requête HTTP
	response, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Erreur lors de la requête HTTP:", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Println("La requête a retourné un code de statut non 200 OK:", response.StatusCode)
		return
	}

	var items []Itemm
	err = json.NewDecoder(response.Body).Decode(&items)
	if err != nil {
		fmt.Println("Erreur lors du décodage JSON:", err)
		return
	}

	// Deuxième requête HTTP
	response2, err := http.Get(apiURL2)
	if err != nil {
		fmt.Println("Erreur lors de la requête HTTP:", err)
		return
	}
	defer response2.Body.Close()

	if response2.StatusCode != http.StatusOK {
		fmt.Println("La requête a retourné un code de statut non 200 OK:", response2.StatusCode)
		return
	}

	var itemR Relation
	err = json.NewDecoder(response2.Body).Decode(&itemR)
	if err != nil {
		fmt.Println("Erreur lors du décodage JSON:", err)
		return
	}

	// Créer une carte pour stocker les informations regroupées par ID
	idInfoMap := make(map[int]struct {
		Itemm
		DatesLocations map[string][]string
	})

	// Regrouper les informations par ID à partir de la première requête
	for _, item := range items {
		idInfoMap[item.Id] = struct {
			Itemm
			DatesLocations map[string][]string
		}{item, nil}
	}

	// Mettre à jour la carte avec les informations de la deuxième requête
	for _, item := range itemR.Index {
		if existingInfo, ok := idInfoMap[item.Id]; ok {
			existingInfo.DatesLocations = item.DatesLocations
			idInfoMap[item.Id] = existingInfo
		} else {
			// Si l'ID n'existe pas encore dans la carte, ajoutez-le
			idInfoMap[item.Id] = struct {
				Itemm
				DatesLocations map[string][]string
			}{Itemm{Id: item.Id}, item.DatesLocations}
		}
	}

	// Gérer les requêtes HTTP
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Extraire le terme de recherche de la requête
		searchTerm := r.URL.Query().Get("search")

		// Filtrer les artistes en fonction du terme de recherche
		filteredArtists := filterArtists(idInfoMap, searchTerm)

		// Rendre la page avec les résultats filtrés
		renderTemplate(w, filteredArtists, searchTerm)
	})

	// Démarrer le serveur HTTP sur le port 8080
	port := 8080
	fmt.Printf("Serveur écoutant sur le port %d...\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Erreur lors du démarrage du serveur:", err)
	}
}

// Fonction pour filtrer les artistes en fonction du terme de recherche
func filterArtists(idInfoMap map[int]struct {
	Itemm
	DatesLocations map[string][]string
}, searchTerm string) map[int]struct {
	Itemm
	DatesLocations map[string][]string
} {
	filteredArtists := make(map[int]struct {
		Itemm
		DatesLocations map[string][]string
	})

	for id, info := range idInfoMap {
		// Filtrer les artistes dont le nom ou d'autres propriétés contiennent le terme de recherche
		if strings.Contains(strings.ToLower(info.Name), strings.ToLower(searchTerm)) {
			filteredArtists[id] = info
		}
	}

	return filteredArtists
}

// Fonction pour rendre le modèle HTML avec les données
func renderTemplate(w http.ResponseWriter, filteredArtists map[int]struct {
	Itemm
	DatesLocations map[string][]string
}, searchTerm string) {
	// Charger le modèle HTML
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Erreur lors du rendu de la page"+err.Error(), http.StatusInternalServerError)
		return
	}

	// Rendre la page HTML avec les données
	err = tmpl.Execute(w, PageVariables{
		FilteredArtists: filteredArtists,
		SearchTerm:      searchTerm,
	})
	if err != nil {
		http.Error(w, "Erreur lors de l'exécution du modèle HTML"+err.Error(), http.StatusInternalServerError)
		return
	}
}
