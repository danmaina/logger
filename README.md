**Description**

logger is a simple `stdout` and `file` logging package

**Usage**

_fetch the package from github_

`go get github.com/danmaina/logger`

_import the package into your project_

`import logger github.com/danmaina/logger`

_set the directory to write logs to_

`logger.SetLogsDirectory("/path/to/directory")`

_***make sure you have sufficient permissions to create the directories if they do not already exist**_

_output some logs to `INFO, ERR, DEBUG, FATAL and TRACE` logs_

`logger.INFO("Something Awesome", varable, "More Stuff to log", "And the List goes on")`

_Enjoy!_