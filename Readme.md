# Arpicee - The Remote Procedure Framework

Arpicee is a remote procedure framework that enables you to
trigger a variety of "jobs" (Github Actions, AWS Lambdas,
AWS SSM Automations, ...) using convenient interfaces (a CLI,
a slackbot, ...)

## Screenshots

The [following Github Workflow](https://github.com/yannh/arpicee-dispatch-workflow/blob/496ccba14b4db22e9fb525651d855dc790e8e1f5/.github/workflows/main.yml) can be used as an Arpicee, and then triggered either through a CLI:

```
export ARPICEE_GH_OWNER=yannh
export ARPICEE_GH_REPO=arpicee-dispatch-workflow
export ARPICEE_GH_WORKFLOW_NAME=hello
export GH_TOKEN=xxx
$ ./bin/dispatch-github-workflow -h
Usage: ./bin/dispatch-github-workflow [OPTION]... [FILE OR FOLDER]...
  -benice
        Be extra nice? (Default: false)
  -h    display help
  -name string
        Hello who?
  -output string
        output type: json or text (default "text")
$ ./bin/dispatch-github-workflow -name Yann -benice true
✓ sayhello
```

or via a Slackbot:

![Slackbot demo](https://github.com/yannh/arpicee/blob/main/assets/slackbot.gif?raw=true)
