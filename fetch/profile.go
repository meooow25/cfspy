package fetch

import "context"

var rankToColor = map[string]int{
	"":                          0x000000,
	"legendary grandmaster":     0x000000,
	"international grandmaster": 0xFF0000,
	"grandmaster":               0xFF0000,
	"international master":      0xFF8C00,
	"master":                    0xFF8C00,
	"candidate master":          0xAA00AA,
	"expert":                    0x0000FF,
	"specialist":                0x03A89E,
	"pupil":                     0x008000,
	"newbie":                    0x808080,
}

// Profile fetches user profile information using the DefaultFetcher.
func Profile(ctx context.Context, match *ProfileURLMatch) (*ProfileInfo, error) {
	return DefaultFetcher.Profile(ctx, match)
}

// Profile fetches user profile information.
func (f *Fetcher) Profile(ctx context.Context, match *ProfileURLMatch) (*ProfileInfo, error) {
	info, err := f.FetchUserInfo(ctx, match.Handle)
	if err != nil {
		// TODO: Maybe handle changed, check redirect?
		return nil, err
	}
	if info.Rank == "" {
		if info.Handle == "MikeMirzayanov" { // TODO: Anyone else?
			info.Rank = "headquarters"
		} else {
			info.Rank = "unrated"
		}
	}
	p := ProfileInfo{
		User:  info,
		Color: rankToColor[info.Rank],
		URL:   match.URL,
	}
	return &p, nil
}
