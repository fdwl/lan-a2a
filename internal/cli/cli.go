package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fdwl/lan-a2a/internal/logger"
)

type Agent interface {
	GetOnlinePeers() []string
	OpenConnection(peerID string) error
	CloseConnection(peerID string) error
	CreateChannel(name string, peerIDs []string) (string, []string, error)
	LeaveChannel(channelID string) error
	ListChannels() ([]interface{}, error)
	SendMessage(channelID, body string) error
	ShareFile(channelID, filePath string) error
	GetAgentInfo(peerID string) (interface{}, error)
	SetProfile(name, description string, skills []string) error
}

type CLI struct {
	agent    Agent
	agentID  string
	profile  string
	reader   *bufio.Reader
	quit     chan struct{}
}

func New(agent Agent, agentID, profile string) *CLI {
	return &CLI{
		agent:   agent,
		agentID: agentID,
		profile: profile,
		reader:  bufio.NewReader(os.Stdin),
		quit:    make(chan struct{}),
	}
}

func (c *CLI) Run() {
	c.printBanner()
	c.printHelp()

	for {
		fmt.Printf("\033[36mlan-a2a>\033[0m ")
		line, err := c.reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		args := parts[1:]

		switch cmd {
		case "help", "h", "?":
			c.printHelp()
		case "status", "st":
			c.cmdStatus()
		case "list", "ls":
			c.cmdList()
		case "info", "i":
			c.cmdInfo(args)
		case "connect", "conn":
			c.cmdConnect(args)
		case "disconnect", "dc":
			c.cmdDisconnect(args)
		case "channels", "ch":
			c.cmdChannels()
		case "create":
			c.cmdCreateChannel(args)
		case "leave":
			c.cmdLeaveChannel(args)
		case "send", "msg":
			c.cmdSend(args)
		case "share":
			c.cmdShareFile(args)
		case "profile":
			c.cmdProfile(args)
		case "quit", "exit", "q":
			close(c.quit)
			return
		default:
			fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", cmd)
		}
	}
}

func (c *CLI) Quit() <-chan struct{} {
	return c.quit
}

func (c *CLI) printBanner() {
	fmt.Println()
	fmt.Println("\033[35m╔══════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[35m║\033[0m  \033[1mLanA2A Interactive CLI\033[0m                        \033[35m║\033[0m")
	fmt.Println("\033[35m║\033[0m  Agent: \033[33m" + c.agentID + "\033[0m                              \033[35m║\033[0m")
	if c.profile != "" {
		fmt.Println("\033[35m║\033[0m  Profile: \033[32m" + c.profile + "\033[0m                      \033[35m║\033[0m")
	}
	fmt.Println("\033[35m╚══════════════════════════════════════════════════╝\033[0m")
}

func (c *CLI) printHelp() {
	fmt.Println()
	fmt.Println("\033[1mCommands:\033[0m")
	fmt.Println("  \033[33mstatus\033[0m, st          Show agent status")
	fmt.Println("  \033[33mlist\033[0m, ls            List online agents")
	fmt.Println("  \033[33minfo\033[0m, i <peer>     Show agent details")
	fmt.Println("  \033[33mconnect\033[0m, conn <id>  Open connection to peer")
	fmt.Println("  \033[33mdisconnect\033[0m, dc <id> Close connection to peer")
	fmt.Println("  \033[33mchannels\033[0m, ch        List joined channels")
	fmt.Println("  \033[33mcreate\033[0m <name> <id>  Create channel with peer")
	fmt.Println("  \033[33mleave\033[0m <channel_id>  Leave a channel")
	fmt.Println("  \033[33msend\033[0m, msg <ch> <msg> Send message to channel")
	fmt.Println("  \033[33mshare\033[0m <ch> <file>   Share file to channel")
	fmt.Println("  \033[33mprofile\033[0m [name] [desc] Show/update profile")
	fmt.Println("  \033[33mhelp\033[0m, h, ?          Show this help")
	fmt.Println("  \033[33mquit\033[0m, exit, q       Exit")
}

func (c *CLI) cmdStatus() {
	peers := c.agent.GetOnlinePeers()
	channels, _ := c.agent.ListChannels()
	fmt.Printf("\n\033[1mAgent Status\033[0m\n")
	fmt.Printf("  ID:       %s\n", c.agentID)
	fmt.Printf("  Online:   %d peers\n", len(peers))
	fmt.Printf("  Channels: %d joined\n", len(channels))
}

func (c *CLI) cmdList() {
	peers := c.agent.GetOnlinePeers()
	if len(peers) == 0 {
		fmt.Println("No online agents")
		return
	}
	fmt.Printf("\n\033[1mOnline Agents (%d):\033[0m\n", len(peers))
	for _, id := range peers {
		fmt.Printf("  %s\n", id)
	}
}

func (c *CLI) cmdInfo(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: info <peer_id>")
		return
	}
	info, err := c.agent.GetAgentInfo(args[0])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("\n\033[1mAgent Info:\033[0m %v\n", info)
}

func (c *CLI) cmdConnect(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: connect <peer_id>")
		return
	}
	if err := c.agent.OpenConnection(args[0]); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Connected to %s\n", args[0])
}

func (c *CLI) cmdDisconnect(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: disconnect <peer_id>")
		return
	}
	if err := c.agent.CloseConnection(args[0]); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Disconnected from %s\n", args[0])
}

func (c *CLI) cmdChannels() {
	channels, err := c.agent.ListChannels()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if len(channels) == 0 {
		fmt.Println("No channels joined")
		return
	}
	fmt.Printf("\n\033[1mChannels (%d):\033[0m\n", len(channels))
	for _, ch := range channels {
		fmt.Printf("  %v\n", ch)
	}
}

func (c *CLI) cmdCreateChannel(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: create <channel_name> <peer_id> [peer_id2 ...]")
		return
	}
	name := args[0]
	peerIDs := args[1:]
	chID, members, err := c.agent.CreateChannel(name, peerIDs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Channel created: %s (members: %v)\n", chID, members)
}

func (c *CLI) cmdLeaveChannel(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: leave <channel_id>")
		return
	}
	if err := c.agent.LeaveChannel(args[0]); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Left channel %s\n", args[0])
}

func (c *CLI) cmdSend(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: send <channel_id> <message>")
		return
	}
	chID := args[0]
	msg := strings.Join(args[1:], " ")
	if err := c.agent.SendMessage(chID, msg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	logger.Info("message sent", "channel", chID, "length", len(msg))
}

func (c *CLI) cmdShareFile(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: share <channel_id> <file_path>")
		return
	}
	chID := args[0]
	filePath := args[1]
	if err := c.agent.ShareFile(chID, filePath); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("File shared to channel %s\n", chID)
}

func (c *CLI) cmdProfile(args []string) {
	if len(args) == 0 {
		fmt.Printf("\n\033[1mProfile:\033[0m\n")
		fmt.Printf("  %s\n", c.profile)
		fmt.Println("\n  Usage: profile <name> [description]")
		return
	}
	name := args[0]
	desc := ""
	if len(args) > 1 {
		desc = strings.Join(args[1:], " ")
	}
	if err := c.agent.SetProfile(name, desc, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Profile updated: name=%s\n", name)
}
