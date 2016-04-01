# pino
Many online communities have moved on to Slack, but many more remain on IRC. *pino* is a small bridge that allows you to use Slack instead of an IRC client!

When users in the configured IRC channels send messages, you will see them in your Slack. And when you send messages through Slack, IRC users will see it in IRC!

## Usage

1. Make a free Slack account, configure a bot integration, and get the API token.
2. `go get` this repository and all of its dependencies:
    ```bash
    $ go get github.com/kennydo/pino
    ```

3. Build the binary:
    ```bash
    $ cd $GOPATH/src/github.com/kennydo/pino
    $ go build cmd/pino.go
    ```

4. Customize the [config](config-example.yaml) to your liking (using your text editor of choice):
    ```bash
    $ cp config-example.yaml config-rizon.yaml
    $ vim config-rizon.yaml
    ```

5. Run *pino*:
    ```bash
    $ ./pino -config config-rizon.yaml
    ```
