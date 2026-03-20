package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	cache := pokecache.NewCache(5 * time.Second)
	config := Config{"https://pokeapi.co/api/v2/location-area/?limit=20", "https://pokeapi.co/api/v2/location-area/?limit=20", cache}
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

type PokeapiArea struct {
	Count    int
	Next     string
	Previous string
	Results  []struct {
		Name string
		URL  string
	}
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
