package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
)

var (
	currentZone   string
	currentNPC    string
	youSay        string
	lastEventTime time.Time
	dialogs       map[string]*QuestDialog
)

// QuestDialog is every NPC dialog
type QuestDialog struct {
	NPCName      string            `yaml:"npcname"`
	CurrentZone  string            `yaml:"currentzone"`
	Conversation map[string]string `yaml:"conversation"`
}

func main() {
	dialogs = make(map[string]*QuestDialog)
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func run() (err error) {
	if len(os.Args) < 2 {
		err = fmt.Errorf("usage: eqlog2lua <file>")
		return
	}

	var d []byte
	//load quests file, if exists
	fi, err := os.Stat("quests.yml")
	if err == nil {
		d, err = ioutil.ReadFile("quests.yml")
		if err != nil {
			err = errors.Wrap(err, "failed to read quests.yml")
			return
		}
		err = yaml.Unmarshal(d, dialogs)
		if err != nil {
			err = errors.Wrap(err, "failed to parse quests.yml")
			return
		}
	}

	filePath := os.Args[1]
	playerName := ""
	if strings.ToLower(filePath) == "-generate" {
		err = doGenerate()
		return
	}

	fi, err = os.Stat(filePath)
	if err != nil {
		return
	}

	if !strings.Contains(fi.Name(), ".txt") {
		err = fmt.Errorf("file %s must contain .txt ending", fi.Name())
		return
	}

	if !strings.Contains(fi.Name(), "eqlog_") {
		err = fmt.Errorf("file %s must contain eqlog_ file prefix", fi.Name())
		return
	}

	playerName = fi.Name()[strings.Index(fi.Name(), "eqlog_")+6:]
	if !strings.Contains(playerName, "_") {
		err = fmt.Errorf("file %s must contain a eqlog_name_ pattern prefix", fi.Name())
		return
	}
	playerName = playerName[0:strings.Index(playerName, "_")]
	fmt.Println("parsing", filePath, "with player", playerName)

	cfg := tail.Config{
		Follow:    true,
		MustExist: true,
		Poll:      true,
		Location: &tail.SeekInfo{
			Offset: fi.Size(),
		},
	}
	t, err := tail.TailFile(filePath, cfg)
	if err != nil {
		err = errors.Wrap(err, "failed to read log")
		return
	}
	for line := range t.Lines {
		err = doParse(line.Text)
		if err != nil {
			fmt.Println("failed to parse line (", line.Text, ") error:", err)
			err = nil
		}
	}
	return
}

func doParse(line string) (err error) {
	//rawLine := line
	tsIndex := strings.Index(line, "]")
	if tsIndex < 0 {
		err = fmt.Errorf("no timestamp ] found")
		return
	}
	//timestamp := line[0:tsIndex]
	line = line[tsIndex+2:]
	fmt.Println(line)
	if strings.Index(line, "You have entered") == 0 {
		currentZone = line[strings.Index(line, "You have entered")+16:]
		echoPrint("current zone changed")
		return
	}
	if strings.Index(line, "You say, '") == 0 {
		youSay = line[strings.Index(line, "You say, '")+10 : len(line)-2]
		if strings.Index(youSay, "Hail, ") == 0 {
			youSay = "hail"
		}
		echoPrint("You said: " + youSay)
		return
	}
	if len(youSay) > 0 && strings.Index(line, "says, '") > 0 {
		npcName := line[0 : strings.Index(line, "says, ")-1]
		currentNPC = npcName
		qd, ok := dialogs[npcName]
		if !ok {
			qd = &QuestDialog{
				NPCName:      npcName,
				Conversation: make(map[string]string),
				CurrentZone:  currentZone,
			}
		}
		qd.Conversation[youSay] = line[strings.Index(line, "says, '")+7 : len(line)-3]

		echoPrint(qd.NPCName + " said: " + qd.Conversation[youSay])
		dialogs[qd.NPCName] = qd
		youSay = ""
		err = saveYaml()
		return
	}
	if len(youSay) > 0 && strings.Index(line, "You have been assigned the task") == 0 {
		if len(currentNPC) == 0 {
			err = fmt.Errorf("no npc set but assigned task")
			return
		}
		qd, ok := dialogs[currentNPC]
		if !ok {
			qd = &QuestDialog{
				NPCName:      currentNPC,
				Conversation: make(map[string]string),
				CurrentZone:  currentZone,
			}
		}
		qd.Conversation[youSay] = line[0 : len(line)-1]

		echoPrint(qd.NPCName + " gave you a task! " + qd.Conversation[youSay])
		dialogs[qd.NPCName] = qd
		youSay = ""
		err = saveYaml()
		return
	}

	return
}

func echoPrint(message string) {
	fmt.Printf("Z:%s|NPC:%s > %s\n", currentZone, currentNPC, message)
}

func saveYaml() (err error) {
	d, err := yaml.Marshal(&dialogs)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal quest dialog")
		return
	}
	err = ioutil.WriteFile("quests.yml", d, 0744)
	if err != nil {
		err = errors.Wrap(err, "Failed to write quests file")
		return
	}
	return
}

func doGenerate() (err error) {

	for _, d := range dialogs {
		if len(d.Conversation) == 0 {
			continue
		}

		data := "function event_say(e)"
		fileNPCName := doFileNpcName(d.NPCName)
		_, err = os.Stat(fileNPCName)
		if err == nil {
			fmt.Println("skipping", fileNPCName, "already exists")
			continue
		}

		fmt.Println("generating", fileNPCName)
		isFirstNote := true
		for youSay, theySay := range d.Conversation {
			if isFirstNote {
				data += fmt.Sprintf("\n	if(e.message:findi(\"%s\")) then", youSay)
				isFirstNote = false
			} else {
				data += fmt.Sprintf("\n	elseif(e.message:findi(\"%s\")) then", youSay)
			}
			if strings.Index(theySay, "You have been assigned the task") == 0 {
				data += fmt.Sprintf("\n		--%s", theySay)
				data += fmt.Sprintf("\n		e.self:Say(\"Unfortunately, I do not yet have this task to give you.\");")
			} else {
				data += fmt.Sprintf("\n		e.self:Say(\"%s\");", doTheySayCleanup(theySay))
			}
		}

		data += "\n	end"
		data += "\nend"
		err = ioutil.WriteFile(fileNPCName, []byte(data), 0744)
		if err != nil {
			fmt.Println("failed to write", fileNPCName, err.Error())
			continue
		}
	}
	return
}

func doTheySayCleanup(in string) (out string) {
	chunk := in
	for {
		fmt.Println("F:", chunk)
		if strings.Index(chunk, "[") < 0 && strings.Index(chunk, "]") < 0 { //bracketed term
			break
		}

		out += chunk[0:strings.Index(chunk, "[")]
		out += fmt.Sprintf("[\".. eq.say_link(\"%s\") ..\"]", chunk[strings.Index(chunk, "[")+1:strings.Index(chunk, "]")])
		chunk = chunk[strings.Index(chunk, "]")+1:]
		fmt.Println("out", out)
		fmt.Println("F afteR:", chunk)
	}
	out = strings.Replace(out, "Xackery", `".. e.other:GetName() .."`, -1)
	out = strings.Replace(out, "Dark Elf", `".. e.other:GetRace() .."`, -1)
	return
}

func doFileNpcName(npcName string) (fileNpcName string) {
	fileNpcName = "#" + strings.Replace(npcName, " ", "_", -1)
	fileNpcName = strings.Replace(fileNpcName, "`", "-", -1)

	fileNpcName += ".lua"
	return
}
