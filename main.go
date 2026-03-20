package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	pokecache "github.com/eschmechel/gopokedex/internal/pokecache"
)

type cliCommand struct {
	name        string
	description string
	callback    func(c *Config, args ...string) error
}

type Config struct {
	Next     string
	Previous string
	cache    *pokecache.Cache
	Pokedex  map[string]Pokemon
}

type Pokemon struct {
	Name           string `json:"name"`
	ID             int    `json:"id"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Effort   int `json:"effort"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Slot int `json:"slot"`
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
}

var supportedCommands = map[string]cliCommand{
	"exit": {
		name:        "exit",
		description: "Exit the pokedex",
		callback:    commandExit,
	},
	"help": {
		name:        "help",
		description: "Print info about the pokedex",
		callback:    commandHelp,
	},
	"map": {
		name:        "map",
		description: "Print 20 location areas",
		callback:    commandMap,
	},
	"mapb": {
		name:        "mapb",
		description: "Print the previous 20 location areas",
		callback:    commandMapb,
	},
	"explore": {
		name:        "explore",
		description: "Find the pokemon in a location area",
		callback:    commandExplore,
	},
	"catch": {
		name:        "catch",
		description: "Catch a pokemon",
		callback:    commandCatch,
	},
	"inspect": {
		name:        "inspect",
		description: "Inspect a pokemon in your pokedex",
		callback:    commandInspect,
	},
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	cache := pokecache.NewCache(5 * time.Second)
	config := Config{"https://pokeapi.co/api/v2/location-area/?limit=20", "https://pokeapi.co/api/v2/location-area/?limit=20", cache, make(map[string]Pokemon)}
	for {
		fmt.Print("Pokedex > ")
		scanned := scanner.Scan()
		if !scanned {
			fmt.Println("Error scanning")
		}
		input := scanner.Text()
		inputs := strings.Fields(strings.ToLower(input))
		if len(inputs) == 0 {
			continue
		}
		cmdKey := inputs[0]
		if cmd, ok := supportedCommands[cmdKey]; ok {
			if cmd.callback != nil {
				if err := cmd.callback(&config, inputs[1:]...); err != nil {
					fmt.Printf("command error: %v\n", err)
				}
			}
		} else {
			fmt.Printf("Unknown command: %s\n", cmdKey)
		}
	}
}

func commandExit(c *Config, _ ...string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(c *Config, _ ...string) error {
	fmt.Println("Welcome to the Pokedex!\n" +
		"Usage: \n\n" +
		"help: Displays a help message\n" +
		"exit: Exit the pokedex")
	return nil
}

func commandMap(c *Config, _ ...string) error {
	err := mapSubCommand(c, "next")
	return err
}

func commandMapb(c *Config, _ ...string) error {
	err := mapSubCommand(c, "previous")
	return err
}

func mapSubCommand(c *Config, choice string) error {
	var url string
	switch choice {
	case "next":
		url = c.Next
	case "previous":
		url = c.Previous
	}

	if url == "" {
		fmt.Println("No previous page")
		return nil
	}

	body, err := fetchURL(c, url)
	if err != nil {
		return err
	}

	type PokeapiArea struct {
		Count    int
		Next     string
		Previous string
		Results  []struct {
			Name string
			URL  string
		}
	}

	areas := PokeapiArea{}
	err = json.Unmarshal(body, &areas)
	if err != nil {
		fmt.Printf("Error Unmarshaling body, error: %v\n", err)
	}
	if len(areas.Results) == 0 {
		fmt.Printf("Error grabbing areas, 0 areas found\n")
		return nil
	}
	c.Next = areas.Next
	c.Previous = areas.Previous

	for _, area := range areas.Results {
		fmt.Println(area.Name)
	}
	return nil
}

func commandExplore(c *Config, args ...string) error {
	if len(args) == 0 {
		fmt.Println("Please provide a location area")
		return nil
	}
	locationArea := args[0]

	URL := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s/", locationArea)

	body, err := fetchURL(c, URL)
	if err != nil {
		return err
	}

	type area struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}
	areas := area{}
	err = json.Unmarshal(body, &areas)
	if err != nil {
		fmt.Printf("Error unmarshaling body, error: %v\n", err)
		return err
	}
	if len(areas.PokemonEncounters) == 0 {
		fmt.Printf("Error grabbing encounters, 0 encounters found\n")
		return nil
	}

	for _, encounter := range areas.PokemonEncounters {
		fmt.Println(encounter.Pokemon.Name)
	}
	return nil
}

func commandCatch(c *Config, args ...string) error {
	if len(args) == 0 {
		fmt.Println("Please provide a pokemon to catch")
		return nil
	}
	pokemon := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s/", pokemon)
	body, err := fetchURL(c, url)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		fmt.Printf("Error grabbing pokemon, 0 bytes found\n")
		return nil
	}

	poke := Pokemon{}
	err = json.Unmarshal(body, &poke)
	if err != nil {
		fmt.Printf("Error unmarshaling body, error: %v\n", err)
		return err
	}
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemon)

	catchProbability := 1.0 / (1.0 + float64(poke.BaseExperience)/100.0)
	if rand.Float64() < catchProbability {
		fmt.Printf("%s was caught!\n", pokemon)
		c.Pokedex[pokemon] = poke
	} else {
		fmt.Printf("%s escaped!\n", pokemon)
	}
	return nil
}

func commandInspect(c *Config, args ...string) error {
	if len(args) == 0 {
		fmt.Println("Please provide a pokemon to inspect")
		return nil
	}
	pokemon := args[0]
	if poke, found := c.Pokedex[pokemon]; found {
		fmt.Printf("Name: %v\nWeight: %v\nStats:\n\t-hp: %v\n\t-attack: %v\n\t-defense: %v\n\t-special-attack: %v\n\t-special-defense: %v\n\t-speed: %v\nTypes:\n", poke.Name, poke.Height, poke.Weight, poke.Stats[0].BaseStat, poke.Stats[1].BaseStat, poke.Stats[2].BaseStat, poke.Stats[3].BaseStat, poke.Stats[4].BaseStat)
		for _, t := range poke.Types {
			fmt.Printf("\t-%v\n", t.Type.Name)
		}
	} else {
		fmt.Printf("you have not caught that pokemon\n")
	}
	return nil
}

func fetchURL(c *Config, url string) ([]byte, error) {
	if cached, found := c.cache.Get(url); found {
		return cached, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error code: %v", res.StatusCode)
	}

	c.cache.Add(url, body)
	return body, nil
}
