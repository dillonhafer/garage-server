# Garage Server

A server for Raspberry Pi to open a garage door. Used by [garage-ios](https://github.com/dillonhafer/garage-ios).

Hardware I used for project:

1. [Magnetic Reed Switch](http://amzn.to/1XuUrV9) (Optional. Used for door status)
2. [Relay Shield Module](http://amzn.to/1NRZf1R)

I really like the above relay because when the power is disconnected and restored *(i.e. power goes out in the middle of the night)* the relay will remain off. That way a power outage won't open your garage door.
## Options

```
  -http string
    	HTTP listen address (e.g. 127.0.0.1:8225)
  -pin int
    	GPIO pin of relay (default 25)
  -status-pin int
    	GPIO pin of reed switch (default 10)
  -cert string
    	TLS certificate path (e.g. /certs/example.com.cert)
  -key string
    	TLS key path (e.g. /certs/example.com.key)
  -version
    	print version and exit
```

*NOTE: Providing a cert and key will infer the use of TLS*

## Installation Instructions

#### Installation Steps Overview:

1. **[Download garage-server](#download-garage-server)**
2. **[Create init.d script](#create-initd-script)**
3. **[Configure init.d script](#configure-initd-script)**

#### Download garage-server

**Install from source**

Make sure [go](https://golang.org/) is installed on your Raspberry Pi and then you can use `go get` for installation:

```bash
go get github.com/dillonhafer/garage-server
```

**Install from binary**

If you don't have/want to setup [go](https://golang.org/) on your Raspberry Pi you can download a pre-built binary. Remember to download the init.d script 😉

Latest binaries available at https://github.com/dillonhafer/garage-server/releases/latest

#### Create init.d script

Simply copy the init.d script from the src directory.

```bash
cp $GOPATH/src/github.com/dillonhafer/garage-server/garage-server.init /etc/init.d/garage-server
```

#### Configure init.d script

The last thing to do is to configure your init.d script to reflect your Raspberry Pi's configuration.

First set the `GARAGE_SECRET` environment variable. This will ensure JSON requests to the server are authenticated. Be sure to use a very random and lengthy secret.

Just un-comment the following line and add your secret in the init.d script:

```bash
# /etc/init.d/garage-server...

# Remember to set a very strong secret token (e.g. ad23384951c79a42b898e273580564d90e4eee22ad2474cf67475f323817a9ed7640a)
# DO NOT USE the above secret. It's an example only.
GARAGE_SECRET=ad23384951c79a42b898e273580564d90e4eee22ad2474cf67475f323817a9ed7640a
```

Other configuration variables to consider are the `HTTP_ADDR` and `PIN`. Use these
to set what address the web server should listen on and what GPIO pin your Raspberry
Pi is configured to use.

```bash
# /etc/init.d/garage-server...

HTTP_ADDR="0.0.0.0:8225"
PIN=25
STATUS_PIN=10
```

Now just install and start the service:

```bash
sudo chmod +x /etc/init.d/garage-server
sudo update-rc.d garage-server defaults
sudo service garage-server start
```

And then verify that it's running:

```bash
sudo lsof -i :8225
```

should return something like:

```bash
COMMAND    PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
garage-se 3401 root    3u  IPv4   9111      0t0  TCP *:8225 (LISTEN)
```

That's it! The server is now setup!

## Updates

You can update your server with the latest binary with the `update` command in the `init.d` script.

You can keep your server automatically up-to-date with cron: 

```bash
@daily /usr/sbin/service garage-server update && /usr/sbin/service garage-server restart
```

## License

   Copyright 2015 Dillon Hafer

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
