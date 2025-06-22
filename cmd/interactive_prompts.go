package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
)

var promptIns *prompt.Prompt
var suggestions = []prompt.Suggest{
	{Text: "file", Description: "file operation"},
	// {Text: "magnet", Description: "magnet operation"},
	// {Text: "offline-task", Description: "offline task operation"},
	{Text: "exit", Description: "exit"},

	// file operation
	{Text: "RemoveEmptyDir", Description: "remove empty dir in a dir"},
	// {Text: "MoveFlattenFiles", Description: "move flatten files"},
	{Text: "SearchAndMoveFiles", Description: "search file in dir and move to new dir"},
}

var LivePrefixState struct {
	LivePrefix string
	IsEnable   bool
}

// var info = color.New(color.FgWhite, color.BgGreen).SprintFunc()

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()

func execFileOperation(cmds []string) {
	if len(cmds) == 0 {
		return
	}
	if cmds[0] == "RemoveEmptyDir" {
		if len(cmds) != 2 {
			fmt.Println("usage: RemoveEmptyDir targetDirId")
			return
		}
		if cmds[1] == "" {
			fmt.Println("targetDirId is empty")
			return
		}
		err := pAgent.RemoveEmptyDir(cmds[1])
		if err != nil {
			fmt.Println(red(err.Error()))
		}
		return
	}
	// if cmds[0] == "MoveFlattenFiles" {
	// 	if len(cmds) < 2 {
	// 		fmt.Printf("usage:  MoveFlattenFiles targetDirId\n\tMoveFlattenFiles targetDirId newDirName\n\tMoveFlattenFiles targetDirId parentDirId newDirName\n")
	// 		return
	// 	}
	// 	if len(cmds) == 2 {
	// 		err := pAgent.MoveFlattenFiles(cmds[1], "", "")
	// 		if err != nil {
	// 			fmt.Println(red(err.Error()))
	// 		}
	// 		return
	// 	}
	// 	if len(cmds) == 3 {
	// 		err := pAgent.MoveFlattenFiles(cmds[1], cmds[1], cmds[2])
	// 		if err != nil {
	// 			fmt.Println(red(err.Error()))
	// 		}
	// 		return
	// 	}
	// 	err := pAgent.MoveFlattenFiles(cmds[1], cmds[2], cmds[3])
	// 	if err != nil {
	// 		fmt.Println(red(err.Error()))
	// 	}
	// 	return
	// }
	if cmds[0] == "SearchAndMoveFiles" {
		if len(cmds) < 4 {
			fmt.Printf("usage:  SearchAndMoveFiles targetDirId distDirId keyword fileType")
			fmt.Println("\tfileType: 0 All, 1 Document, 2 Image, 3 Audio, 4 Video, 5 Archive, 6 Sofrwore")
			return
		}
		fileType := 0
		if len(cmds) == 4 {
			err := pAgent.SearchAndMoveFiles(cmds[1], cmds[2], cmds[3], fileType)
			if err != nil {
				fmt.Println(red(err.Error()))
			}
			return
		}
		fileType, _ = strconv.Atoi(cmds[4])
		err := pAgent.SearchAndMoveFiles(cmds[1], cmds[2], cmds[3], fileType)
		if err != nil {
			fmt.Println(red(err.Error()))
		}
		return
	}
	fmt.Println("unknown commands:", red(strings.Join(cmds, " ")))
	fmt.Printf("usage: %s targetDirId\n       %s targetDirId newDirName\n", yellow("RemoveEmptyDir"), yellow("SearchAndMoveFiles"))
}

func executor(in string) {
	blocks := strings.Fields(in)
	if len(blocks) == 0 {
		return
	}
	switch blocks[0] {
	case "exit":
		fmt.Println("Bye!")
		os.Exit(0)
	case "file":
		LivePrefixState.LivePrefix = "file> "
		LivePrefixState.IsEnable = true
		execFileOperation(blocks[1:])
		return
		// case "magnet":
		// 	LivePrefixState.LivePrefix = "magnet> "
		// 	LivePrefixState.IsEnable = true
		// 	return
		// case "offline-task":
		// 	LivePrefixState.LivePrefix = "offline-task> "
		// 	LivePrefixState.IsEnable = true
		// 	return
	}
	if strings.Contains(LivePrefixState.LivePrefix, "file> ") {
		execFileOperation(blocks)
		return
	}
	// if strings.Contains(LivePrefixState.LivePrefix, "magnet> ") {
	// 	return
	// }
	// if strings.Contains(LivePrefixState.LivePrefix, "offline-task> ") {
	// 	return
	// }
}

func completer(in prompt.Document) []prompt.Suggest {
	w := in.GetWordBeforeCursor()
	if w == "" {
		return []prompt.Suggest{}
	}
	return prompt.FilterHasPrefix(suggestions, w, true)
}

func changeLivePrefix() (string, bool) {
	return LivePrefixState.LivePrefix, LivePrefixState.IsEnable
}

func startPrompts() {
	promptIns = prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(">>> "),
		prompt.OptionLivePrefix(changeLivePrefix),
		prompt.OptionTitle("rss2cloud"),
	)
	promptIns.Run()
}
