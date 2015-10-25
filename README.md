# Garage Server

A server for Raspberry Pi to open a garage door. Used by [garage-ios](https://github.com/dillonhafer/garage-ios).

## Options

```
  -http string
    	HTTP listen address (e.g. 127.0.0.1:8225)
  -pin int
    	GPIO pin of relay (default 25)
  -version
    	print version and exit
```

## Installation Instructions

#### Installation Steps Overview:

1. Download garage-server
2. Create init.d script
3. Configure init.d script

#### 1. Download garage-server

**Install from source**

Make sure [go](https://golang.org/) is installed on your Raspberry Pi and then you can use `go get` for installation:

```bash
go get github.com/dillonhafer/garage-server
```

**Install from binary**

If you don't have/want to setup [go](https://golang.org/) on your Raspberry Pi you can download a pre-built binary. Remember to download the init.d script 😉

Binaries available at https://github.com/dillonhafer/garage-server/releases

#### 2. Create init.d script

Simply copy the init.d script from the src directory.

```bash
cp $GOPATH/src/github.com/dillonhafer/garage-server/garage-server.init /etc/init.d/garage-server
```

#### 3. Configure init.d script

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
