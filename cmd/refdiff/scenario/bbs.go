package scenario

import (
	"context"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The community board scenario of Plan 14 M11o.

func init() { scenarios = append(scenarios, boardScenarios...) }

// boardScenarios are the community board scenarios.
var boardScenarios = []Scenario{
	{
		Name: "bbs-home",
		Description: "Alt+B: the home page in its ShowBoard parts 101-103 (it counts the server's clans: both servers need as many), " +
			"then the memo page and the region list of the nine castles (no castle owned on either server)",
		Roles:  []string{"alice"},
		Run:    bbsHome,
		Ignore: npcNoise,
	},
}

// awaitBoardEnd waits for the last part (103) of a board page.
func awaitBoardEnd(ctx context.Context, p *Player) error {
	_, err := testclient.Await(ctx, p.Game(), func(b testclient.ShowBoard) bool {
		id, _ := b.Part()
		return id == "103"
	})
	return err
}

// boardBypass is a step that clicks a board link and waits for the page.
func boardBypass(p *Player, cmd string) Step {
	return Step{p.Role + " opens " + cmd, func(ctx context.Context) error {
		if err := p.Game().Bypass(ctx, cmd); err != nil {
			return err
		}
		return awaitBoardEnd(ctx, p)
	}}
}

func bbsHome(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	return s.Steps(ctx,
		Idle(time.Second),
		Step{"alice presses Alt+B", func(ctx context.Context) error {
			if err := a.Game().RequestShowBoard(ctx); err != nil {
				return err
			}
			return awaitBoardEnd(ctx, a)
		}},
		boardBypass(a, "_bbsmemo"),
		boardBypass(a, "_bbsloc"),
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}
