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
	callback    func(c *Config) error
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
	"explore": cliCommand{
		name:				 "explore",
		decription: "Find the pokemon in a location area",
		callback: commandExplore,
	},
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	cache := pokecache.NewCache(5 * time.Millisecond)
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
				if err := cmd.callback(&config); err != nil {
					fmt.Printf("command error: %v\n", err)
				}
			}
		} else {
			fmt.Printf("Unknown command: %s\n", cmdKey)
		}
	}
}

func commandExit(c *Config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(c *Config) error {
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

func commandMap(c *Config) error {
	err := mapSubCommand(c, "next")
	return err
}

func commandMapb(c *Config) error {
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
	err := json.Unmarshal(body, &areas)
	if err != nil {
		fmt.Printf("Error Unmarshaling body, error: %v\n", err)
	}
	if len(areas.Results) == 0 {
		fmt.Printfi("Error grabbing areas, 0 areas found\n")
		return nil
	}
	c.Next = areas.Next
	c.Previous = areas.Previous

	for _, area := range areas.Results {
		fmt.Println(area.Name)
	}
	return nil
}

func commandExplore(c *Config, locationArea string) error {
	
	if locationArea == "" {
		fmt.Println("Please provide a location area")
		return nil
	}

	URL := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s/", locationArea)
	
	body, err := fetchURL(c,URL)
	if err != nil {
		return err
	}
	var area struct {
		pokemon_encounters []struct {
			pokemon struct {
				name string
				url string
			}
		}
	}
	
	for _, encounter := range area.pokemon_encounters {
		fmt.Println(encounter.pokemon.name)
	}
	return nil
}

func fetchURL(c *Config, url string) ([]byte, error) {

	if cached []byte, found bool := c.cache.Get(URL)
		body = cached
	} else {
		res *http.Response, err := http.Get(URL)
		if err != nil {
			return nil,err
		}
		//ignore error
		defer func() { _ = res.Body.Close() }()
		
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil,err
		}

		if res.StatusCode != http.StatusOK {
			return nil,fmt.Errorf("error code: %v", res.StatusCode)
		}

		c.cache.Add(URL, body)
	}

	return body, nil
}
