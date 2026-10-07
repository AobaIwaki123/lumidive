// Package ticketdive provides client and parser capabilities for TicketDive events.
package ticketdive

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
)

var nextDataRegex = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

type rawNextData struct {
	BuildID string `json:"buildId"`
	Props   struct {
		PageProps struct {
			ServerTime     int64          `json:"serverTime"`
			SuperJSONProps struct {
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
	Event          rawEvent        `json:"event"`
	Stages         []rawStage      `json:"stages"`
	TicketInfoList []rawTicketInfo `json:"ticketInfoList"`
}

type rawEvent struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Detail  *string `json:"detail"`
	Release *string `json:"release"`
}

type rawStage struct {
	ID         string      `json:"id"`
	StageName  string      `json:"stageName"`
	StartStage *string     `json:"startStage"`
	OpenVenue  *string     `json:"openVenue"`
	Venue      rawVenue    `json:"venue"`
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
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	ReceptionType       string          `json:"receptionType"`
	StartApply          *string         `json:"startApply"`
	EndApply            *string         `json:"endApply"`
	PaymentChannels     []string        `json:"paymentChannels"`
	TransferRestriction *string         `json:"transferRestriction"`
	Status              *string         `json:"status"`
	Customize           []rawCustomize  `json:"customize"`
	TicketTypes         []rawTicketType `json:"ticketTypes"`
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

// ParseResult holds the parsed event and upstream source metadata.
type ParseResult struct {
	Event  *api.Event
	Source *api.SourceMetadata
}

// ParseHTML parses TicketDive event HTML and returns a ParseResult.
func ParseHTML(html string, norm *NormalizedURL) (*ParseResult, error) {
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

	event := buildEventModel(superJSON.EventDetail, superJSON.EventImages, norm)

	platform := "ticketdive"
	now := time.Now().UTC()
	source := &api.SourceMetadata{
		Platform:   &platform,
		Url:        &norm.CanonicalURL,
		BuildId:    &data.BuildID,
		FetchedAt:  &now,
		ServerTime: &data.Props.PageProps.ServerTime,
	}

	return &ParseResult{
		Event:  event,
		Source: source,
	}, nil
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
				artistURL := fmt.Sprintf("https://ticketdive.com/artist/%s", art.ID)
				stArtists = append(stArtists, api.Artist{
					Id:   art.ID,
					Name: art.Name,
					Url:  &artistURL,
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
			artistURL := fmt.Sprintf("https://ticketdive.com/artist/%s", id)
			allArtists = append(allArtists, api.Artist{
				Id:   id,
				Name: name,
				Url:  &artistURL,
			})
		}
		ev.Artists = &allArtists
	}

	// Ticket Groups and Stats accumulation
	var totalTickets int
	var soldOutTickets int
	minPrice := -1
	maxPrice := -1

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

		if tg.TransferRestriction != nil && *tg.TransferRestriction != "" {
			restr := []string{*tg.TransferRestriction}
			groupItem.Restrictions = &restr
		}

		// Customize questions
		if len(tg.Customize) > 0 {
			custQuestions := make([]api.CustomizeQuestion, 0, len(tg.Customize))
			for idx, c := range tg.Customize {
				qID := fmt.Sprintf("q_%d", idx)
				q := api.CustomizeQuestion{
					Id:       &qID,
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
			totalTickets++
			isSoldOut := false
			if (tt.RemainingRate != nil && *tt.RemainingRate == 0) || (tt.Status != nil && *tt.Status == "closed") {
				isSoldOut = true
				soldOutTickets++
			}

			if minPrice == -1 || tt.Price < minPrice {
				minPrice = tt.Price
			}
			if maxPrice == -1 || tt.Price > maxPrice {
				maxPrice = tt.Price
			}

			curr := "JPY"
			var totalPrice *int
			if tt.Fee != nil {
				tp := tt.Price + *tt.Fee
				totalPrice = &tp
			}

			typeItem := api.TicketType{
				Id:             tt.ID,
				Name:           tt.Name,
				Detail:         tt.Detail,
				Currency:       &curr,
				Price:          tt.Price,
				Fee:            tt.Fee,
				TotalPrice:     totalPrice,
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

	// Stats
	hasAvailable := (totalTickets > soldOutTickets)
	if minPrice == -1 {
		minPrice = 0
	}
	if maxPrice == -1 {
		maxPrice = 0
	}
	ev.Stats = &api.EventStats{
		TotalTicketTypes:    &totalTickets,
		SoldOutTicketTypes:  &soldOutTickets,
		MinPrice:            &minPrice,
		MaxPrice:            &maxPrice,
		HasAvailableTickets: &hasAvailable,
	}

	return ev
}
