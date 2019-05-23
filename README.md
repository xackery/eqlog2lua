# eqlog2lua
Generate quest lua from live eqlog

[Download is found in releases](https://github.com/xackery/eqlog2lua/releases)

usage: eqlog2lua.exe <file>

e.g. eqlog2lua.exe eqlog_Player_Server.txt

There is two steps to the process.
First, it generates a quests.yml file. This is reloaded any time the program restarts.
Second, it creates lua files based on quests.yml data.


example quests.yml:
```yaml
Talya Darkfall:
  npcname: Talya Darkfall
  currentzone: ""
  conversation:
    creatures: I was under disguise in Neriak when Tani N`Mar tasked me out with this
      detail  If I had refused it I might not have escaped out of Neriak.  I decided
      to follow through on the [tasks] to prevent them from finding out my identity.
    goblins: You have been assigned the task 'Scouting the Goblins'.
    hail: It took you long enough to get here and you don't look like much of a scout.  You
      should do fine for the [creatures] that we are scouting.
    shadowed men: You have been assigned the task 'Lurking in the Shadows'.
    tasks: Our tasks are to find out as much information about the creatures in Lavastorm
      as we can.  I have been scouting [shadowed men], [lavaspinners], [war drakes],
      and [goblins].

```

second step is not yet coded. Point is to generate cookie cutter code for lua stuff to bootstrap process
