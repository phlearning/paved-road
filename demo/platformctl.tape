# Terminal demo of the golden path, recorded with VHS against a running platform:
#   cd paved-road && make cli && vhs demo/platformctl.tape
Output demo/platformctl.gif

Require bin/platformctl

Set Shell bash
Set FontSize 15
Set Width 1280
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms
Set PlaybackSpeed 1.0

Hide
Type "export PS1='\[\e[1;36m\]paved-road\[\e[0m\] $ ' PATH=$PWD/bin:$PATH && rm -rf apps/payments-api && clear"
Enter
Show

Type "# Is the platform healthy?"
Sleep 400ms
Enter
Type "platformctl doctor"
Sleep 400ms
Enter
Sleep 4s

Type "# Which services run, and are they in sync with Git?"
Enter
Type "platformctl status"
Sleep 400ms
Enter
Sleep 4s

Type "clear"
Enter
Type "# A new service from the golden path: code, tests, Dockerfile, values"
Enter
Type "platformctl new-service payments-api --lang go"
Sleep 400ms
Enter
Sleep 3s

Type "cat apps/payments-api/values.yaml"
Sleep 400ms
Enter
Sleep 4s

Type "clear"
Enter
Type "# Ask the platform assistant: a RAG over docs/, in French or English"
Enter
# VHS cannot type accented characters: the question avoids them.
Type `platformctl ask "Comment ajouter une base PostgreSQL pour mon service ?"`
Sleep 500ms
Enter
Sleep 22s

Hide
Type "rm -rf apps/payments-api"
Enter
Show
Sleep 1s
