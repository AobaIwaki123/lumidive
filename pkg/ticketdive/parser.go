// Package ticketdive provides client and parser capabilities for TicketDive events.
package ticketdive

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/AobaIwaki123/lumidive/pkg/api"
)

var nextDataRegex = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

type rawNextData struct {
	BuildID string `json:"buildId"`
	Props   struct {
		PageProps struct {
			ServerTime       int64 `json:"serverTime"`
			SuperJSONProps   struct {
				JSON rawSuperJSON `json:"json"`
			} `json:"__superjsonProps"`
		} `json:"pageProps"`
	} `json:"props"`
}

type rawSuperJSON struct {
	EventDetail *rawEventDetail `json:"eventDetail"`
	EventImages []struct {
		ImageSource string `json:"imageSource"`
	} `json:"eventImages"`
}

type rawEventDetail struct {
	Event          rawEvent         `json:"event"`
	Stages         []rawStage       `json:"stages"`
	TicketInfoList []rawTicketInfo  `json:"ticketInfoList"`
}

type rawEvent struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Detail  *string `json:"detail"`
	Release *string `json:"release"`
}

type rawStage struct {
	ID         string     `json:"id"`
	StageName  string     `json:"stageName"`
	StartStage *string    `json:"startStage"`
	OpenVenue  *string    `json:"openVenue"`
	Venue      rawVenue   `json:"venue"`
	Artists    []rawArtist `json:"artists"`
}

type rawVenue struct {
	Name    *string `json:"name"`
	Address *string `json:"address"`
}

type rawArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawTicketInfo struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	ReceptionType   string              `json:"receptionType"`
	StartApply      *string             `json:"startApply"`
	EndApply        *string             `json:"endApply"`
	PaymentChannels []string            `json:"paymentChannels"`
	Status          *string             `json:"status"`
	Customize       []rawCustomize      `json:"customize"`
	TicketTypes     []rawTicketType     `json:"ticketTypes"`
}

type rawCustomize struct {
	Label         *string `json:"label"`
	Required      *bool   `json:"required"`
	Type          *string `json:"type"`
	SelectOptions []struct {
		Value string `json:"value"`
	} `json:"selectOptions"`
}

type rawTicketType struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Detail         *string  `json:"detail"`
	Price          int      `json:"price"`
	Fee            *int     `json:"fee"`
	RemainingRate  *float32 `json:"remainingRate"`
	Status         *string  `json:"status"`
	MaxNumPerApply *int     `json:"maxNumPerApply"`
	Prefix         *string  `json:"prefix"`
}

// ParseHTML parses TicketDive event HTML and returns an api.Event.
func ParseHTML(html string, norm *NormalizedURL) (*api.Event, error) {
	match := nextDataRegex.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("could not find __NEXT_DATA__ in response HTML")
	}

	var data rawNextData
	if err := json.Unmarshal([]byte(match[1]), &data); err != nil {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__ json: %w", err)
	}

	superJSON := data.Props.PageProps.SuperJSONProps.JSON
	if superJSON.EventDetail == nil {
		return nil, fmt.Errorf("eventDetail not found in page data")
	}

	return buildEventModel(superJSON.EventDetail, superJSON.EventImages, norm), nil
}

func buildEventModel(detail *rawEventDetail, images []struct {
	ImageSource string `json:"imageSource"`
}, norm *NormalizedURL) *api.Event {
	ev := &api.Event{
		Id:       detail.Event.ID,
		Slug:     norm.EventID,
		Name:     detail.Event.Name,
		Detail:   detail.Event.Detail,
		Release:  detail.Event.Release,
		Url:      norm.CanonicalURL,
		ShortUrl: norm.ShortURL,
	}

	// Images
	if len(images) > 0 {
		imgList := make([]string, 0, len(images))
		for _, img := range images {
			if img.ImageSource != "" {
				imgList = append(imgList, img.ImageSource)
			}
		}
		ev.Images = &imgList
	}

	// Stages & Artists map
	artistMap := make(map[string]string)
	stagesList := make([]api.Stage, 0, len(detail.Stages))
	for _, st := range detail.Stages {
		stageItem := api.Stage{
			Id:        st.ID,
			StageName: st.StageName,
			StartAt:   st.StartStage,
			OpenAt:    st.OpenVenue,
			Venue: &api.Venue{
				Name:    st.Venue.Name,
				Address: st.Venue.Address,
			},
		}

		if len(st.Artists) > 0 {
			stArtists := make([]api.Artist, 0, len(st.Artists))
			for _, art := range st.Artists {
				stArtists = append(stArtists, api.Artist{
					Id:   art.ID,
					Name: art.Name,
				})
				artistMap[art.ID] = art.Name
			}
			stageItem.Artists = &stArtists
		}
		stagesList = append(stagesList, stageItem)
	}
	ev.Stages = &stagesList

	// Artists
	if len(artistMap) > 0 {
		allArtists := make([]api.Artist, 0, len(artistMap))
		for id, name := range artistMap {
			allArtists = append(allArtists, api.Artist{
				Id:   id,
				Name: name,
			})
		}
		ev.Artists = &allArtists
	}

	// Ticket Groups
	groupsList := make([]api.TicketGroup, 0, len(detail.TicketInfoList))
	for _, tg := range detail.TicketInfoList {
		groupItem := api.TicketGroup{
			Id:            tg.ID,
			Name:          tg.Name,
			ReceptionType: tg.ReceptionType,
			StartApply:    tg.StartApply,
			EndApply:      tg.EndApply,
			Status:        tg.Status,
		}

		if len(tg.PaymentChannels) > 0 {
			ch := append([]string{}, tg.PaymentChannels...)
			groupItem.PaymentChannels = &ch
		}

		// Customize questions
		if len(tg.Customize) > 0 {
			custQuestions := make([]api.CustomizeQuestion, 0, len(tg.Customize))
			for _, c := range tg.Customize {
				q := api.CustomizeQuestion{
					Label:    c.Label,
					Required: c.Required,
					Type:     c.Type,
				}
				if len(c.SelectOptions) > 0 {
					opts := make([]string, 0, len(c.SelectOptions))
					for _, o := range c.SelectOptions {
						opts = append(opts, o.Value)
					}
					q.Options = &opts
				}
				custQuestions = append(custQuestions, q)
			}
			groupItem.Customize = &custQuestions
		}

		// Ticket Types
		typesList := make([]api.TicketType, 0, len(tg.TicketTypes))
		for _, tt := range tg.TicketTypes {
			isSoldOut := false
			if (tt.RemainingRate != nil && *tt.RemainingRate == 0) || (tt.Status != nil && *tt.Status == "closed") {
				isSoldOut = true
			}

			typeItem := api.TicketType{
				Id:             tt.ID,
				Name:           tt.Name,
				Detail:         tt.Detail,
				Price:          tt.Price,
				Fee:            tt.Fee,
				RemainingRate:  tt.RemainingRate,
				IsSoldOut:      isSoldOut,
				Status:         tt.Status,
				MaxNumPerApply: tt.MaxNumPerApply,
				Prefix:         tt.Prefix,
			}
			typesList = append(typesList, typeItem)
		}
		groupItem.Types = typesList
		groupsList = append(groupsList, groupItem)
	}
	ev.TicketGroups = &groupsList

	return ev
}
