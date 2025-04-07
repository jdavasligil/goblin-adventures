# Goblin Adventures
A bot that runs the game 'Goblin Adventures', a text based coop RPG on Twitch.


## Environment Variables
Create a file named `.env` and supply the environment variables:
```
CLIENT_ID=<Copy ID Here>
CLIENT_SECRET=<Copy Secret Here>
BOT_USER_ID=<Copy Bot ID Here>
BROADCASTER_ID=<Copy Broadcaster ID here>
```

## Build
Generate TLS certs by running `./scripts/make_ssl_keys.sh` in the terminal from
the project root directory.

Run the bot with `go run .`


# Design
Text based party dungeon crawler.

## Pillars

### Simple & Accessible
The game needs to be easy to play with very few text inputs required.

### Quick & Satisfying
We want a fairly quick and satisfying feedback loop. Go into dungeon, get shinies,
return to Goblin Town. Rewards should make your goblin more powerful and able to 
last longer / go deeper.

### Risk & Reward
The temptation to explore just one more room should inspire boldness and fear alike.
A powerful item or valuable treasure could be just around the corner, but so could
certain doom.

### Cooperative Play
Teamwork is encouraged. Actions should benefit other players. Players may vote on
where to go next.

## Gameplay
Viewers are goblins hanging out in Goblin Town. They accrue shinies by delving 
into The Dungeons of Chaos.

There is only ever one party. Once the delve begins, the party is locked until
either everyone has voted to return or died.

### Main Loop
1. Join or start the dungeon party with `!join`. The first to join is leader.
2. When the leader is ready they may `!begin` the delve. The delve begins automatically if the party is full or after 10 seconds.
3. A random seed is used to procedurally generate the dungeon as the players explore.
4. The players will descend stairs and enter the first room of the dungeon. If a monster is present, combat begins. If treasure is present, it is automatically divided among the party.
5. After the room event concludes (combat, trap, treasure, encounter, etc.), players vote on where to go next. Enter `!home` to return to Goblin Town. `!north/south/east/west` are valid directions.
6. Sometimes stairs are found which go to lower levels. Lower levels are more dangerous.
7. After returning, XP is awarded based on the treasure obtained and rooms survived.


### Combat
If a monster is present in a room and aware of the party, it will attack and 
initiate combat. Otherwise, the party has the option to `!sneak` by, `!steal`, or `!attack`.

Combat progresses in rounds until the monster (or party) are either defeated or flee.

Initiative goes by sides (party vs monster). Unaware targets lose automatically.

Each round sides take turn with actions. Party members may enter `!flee` to run away into a random room on the next turn, `!use` to use a previously `!ready [item]` readied item or `!use [item]`, `!cast` to begin casting a previously `!prepare [spell]` prepared spell or `!cast [spell]`, `!attack` to make a melee attack, `!shoot` to make a ranged attack if a ranged weapon is equipped, and `!taunt` to goad the target to attack them.

Once all members have input their actions, or after 6 seconds, the turn ends. The default action is attack.

Damage and effects are applied. The monster will check for morale, then decide an action.

If a lone monster attempts to flee, players get one round to act before it is gone on its next turn.

## Death
When a goblin reaches 0 HP, they are in a vulnerable state. Each attack reduces
their Might stat directly, requiring a Might check vs death. At 0 Might the goblin dies.

### Recovery
Stats and HP recover completely when returning home.

### Rest
Any time out of combat, players can choose to `!rest` and recover their HP.
However, there is always a chance that a wandering monster may attack and surprise
the party during their rest.

## Items
Various useful items may be found in dungeons, typically magical ones which
have a single use and create an effect. Many items are unidentified until used,
at which point they are known.

Goblins can carry a maximum of 5 unequipped items on their person.

Items can be checked with `!inventory` and dropped with `!drop`.

Items nearby can be searched with `!search` and picked up with `!grab`.

Items can be given to another player directly with `!give [item] [player]`.

### Arms and Armor
Goblins can `!equip [armor/weapon]` a single armor and weapon. See what a goblin has equipped with `!inspect [name]`.
Armor reduces damage taken, weapons deal damage. Simple. Attacks are either melee, ranged, or magical.

Like in Runescape, different armors and weapons provide tradeoffs vs each other.

## Rewards
- Goblins earn XP which automatically levels them up and improves goblin skills.
- Shinies are goblin currency, which can be used to gamble and purchase items.
- Items provide stat and skill bonuses. Some are consumable.

### Treasure
Treasure comes in the form of `shinies` which represent coins, gems, baubles, and various trinkets.

### Experience Points (XP)
`1 shiny = 1 xp`. Bonus XP is awarded for various actions: defeating monsters, sneaking past, stealing, etc.
XP is only awarded upon returning home, and is equally distributed to each member.

At certain thresholds, goblins will level up gaining increased hit protection,
improving stats and skills. Stats and skills improve semi-randomly, with lower ones
increasing more often and influenced by birth sign.


## Random Ideas
Fuzzy matching / error correcting to estimate the closest intention of the writer
given imperfect text
