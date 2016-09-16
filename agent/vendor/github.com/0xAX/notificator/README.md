notificator
===========================

Desktop notification with golang for:

  * Windows with `growlnotify`;
  * Mac OS X with `growlnotify`;
  * Linux with `notify-send` for gnome and `kdialog` for kde.

Usage
------

```go
package main

import (
  "github.com/0xAX/notificator"
)

var notify *notificator.Notificator

func main() {

  notify = notificator.New(notificator.Options{
    DefaultIcon: "icon/default.png",
    AppName:     "My test App",
  })

  notify.Push("title", "text", "/home/user/icon.png", notificator.UR_CRITICAL)
}
```

TODO
-----

  * Add more options for different notificators.

Сontribution
------------

  * Fork;
  * Make changes;
  * Send pull request;
  * Thank you.

author
----------

[@0xAX](https://twitter.com/0xAX)
