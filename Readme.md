# Arpicee - The Remote Procedure Framework

Arpicee is a remote procedure framework that enables you to
trigger a variety of "jobs" (Github Actions, AWS Lambdas,
AWS SSM Automations, ...) using convenient interfaces (a CLI,
a slackbot, ...)

## Screenshots

The [following Github Workflow] can be used as an Arpicee, and then triggered either through a CLI:

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

![Select Arpicee](https://github.com/yannh/arpicee/blob/main/assets/select_arpicee.png?raw=true)
![Configure Arpicee](https://github.com/yannh/arpicee/blob/main/assets/configure_arpicee.png?raw=true)
![Arpicee result](https://github.com/yannh/arpicee/blob/main/assets/arpicee_result.png?raw=true)
