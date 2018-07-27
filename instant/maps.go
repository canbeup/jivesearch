package instant

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/jivesearch/jivesearch/instant/location"
	"golang.org/x/text/language"
)

// Maps is an instant answer
type Maps struct {
	LocationFetcher location.Fetcher
	Answer
}

// Map is a instant answer response
type Map struct {
	location.Location
	Directions  bool
	Origin      string
	Destination string
}

func (m *Maps) setQuery(req *http.Request, q string) Answerer {
	m.Answer.setQuery(req, q)
	return m
}

func (m *Maps) setUserAgent(req *http.Request) Answerer {
	return m
}

func (m *Maps) setLanguage(lang language.Tag) Answerer {
	m.language = lang
	return m
}

func (m *Maps) setType() Answerer {
	m.Type = "maps"
	return m
}

func (m *Maps) setRegex() Answerer {
	m.regex = append(m.regex, regexp.MustCompile(`^directions to (?P<end>.*)$`))
	m.regex = append(m.regex, regexp.MustCompile(`^directions (?P<start>.*) to (?P<end>.*)$`))

	triggers := []string{
		"map", "maps", "direction", "directions",
	}

	t := strings.Join(triggers, "|")
	m.regex = append(m.regex, regexp.MustCompile(fmt.Sprintf(`^(?P<trigger>%s)$`, t)))
	m.regex = append(m.regex, regexp.MustCompile(fmt.Sprintf(`^(?P<trigger>%s) .*$`, t)))
	m.regex = append(m.regex, regexp.MustCompile(fmt.Sprintf(`^.* (?P<trigger>%s)$`, t)))

	return m
}

func (m *Maps) solve(r *http.Request) Answerer {
	// The caller is expected to provide the solution when triggered, preferably in JavaScript
	ip := getIPAddress(r)

	city, err := m.LocationFetcher.Fetch(ip)
	if err != nil {
		m.Err = err
	}

	mm := Map{
		Location: location.Location{
			Latitude:  city.Location.Latitude,
			Longitude: city.Location.Longitude,
		},
	}

	if m.triggerWord == "direction" || m.triggerWord == "directions" {
		mm.Directions = true
	}

	mm.Origin = m.remainderM["start"]
	var ok bool

	if mm.Destination, ok = m.remainderM["end"]; ok {
		mm.Directions = true
		if mm.Origin == "" { // if no starting point then use their current location
			if c, ok := city.City.Names["en"]; ok {
				if len(city.Subdivisions) > 0 {
					if s, ok := city.Subdivisions[0].Names["en"]; ok {
						mm.Origin = fmt.Sprintf("%v, %v", c, s)
					}
				}
			}
		}
	}

	m.Data.Solution = mm

	return m
}

func (m *Maps) setCache() Answerer {
	m.Cache = true
	return m
}

func (m *Maps) tests() []test {
	typ := "maps"

	tests := []test{
		{
			query: "map",
			ip:    net.ParseIP("161.59.224.138"),
			expected: []Data{
				{
					Type:      typ,
					Triggered: true,
					Solution: Map{
						Location: location.Location{Latitude: 12, Longitude: 18},
					},
					Cache: true,
				},
			},
		},
		{
			query: "directions",
			ip:    net.ParseIP("161.59.224.138"),
			expected: []Data{
				{
					Type:      typ,
					Triggered: true,
					Solution: Map{
						Location:   location.Location{Latitude: 12, Longitude: 18},
						Directions: true,
					},
					Cache: true,
				},
			},
		},
		{
			query: "directions to new york city",
			ip:    net.ParseIP("161.59.224.138"),
			expected: []Data{
				{
					Type:      typ,
					Triggered: true,
					Solution: Map{
						Location:    location.Location{Latitude: 12, Longitude: 18},
						Directions:  true,
						Origin:      "",
						Destination: "new york city",
					},
					Cache: true,
				},
			},
		},
	}

	return tests
}
