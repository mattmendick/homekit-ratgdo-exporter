# homekit-ratgdo-exporter

If you have a ratgdo garage door controller and have installed https://github.com/ratgdo/homekit-ratgdo on it, you can use this Prometheus exporter to get the status of the controller and the door/light.

It's based polling the `status.json` endpoint exposed by the controller.

## Building
Clone the repo and build with `go build`

## Running
The --help parameter will print
```
./homekit-ratgdo-exporter --help
Usage of ./homekit-ratgdo-exporter:
  -json-address string
    	The address of the JSON endpoint (default "http://ratgdo/status.json")
  -location string
    	The location label for the metrics (default "home")
  -port string
    	The port to expose metrics on (default "8080")
```

I run it like this:
`./homekit-ratgdo-exporter -port 9987 -json-address "http://10.10.10.10/status.json"`

It's helpful to give your ratgdo controller the same IP address or a name on your network.

## AI written
I used ChatGPT to help write this, so if there's anything wonky about it or a bit strange, perhaps that's why. PRs welcome if for some reason you come across this repo.