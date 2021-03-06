package hub

import (
    "net"
    "bufio"
    "fmt"
    "os"
    "time"
    "bytes"
    util "../../util"
)

type Client struct {
    in chan []byte
    out chan []byte
    user_id uint64
    reader *bufio.Reader
    writer *bufio.Writer
}

type Hub struct {
    in chan []byte
    out chan []byte
    connections chan net.Conn
    clients []*Client
    writer *bufio.Writer
}

/**
 * The function reads data from the client.
 */
func (client *Client) Read() {
    for {
        line, _ := client.reader.ReadBytes('\n')
        client.in <- line
    }
}

/**
 * The function sends data to the client.
 */
func (client *Client) Send() {
    for data := range client.out {
        for _, b := range data {
            client.writer.WriteByte(b)
        }
        client.writer.Flush()
    }
}

/**
 * The function joins client to the hub.
 * @param net.Conn connection
 */
func (hub *Hub) Join(connection net.Conn) {
    client := InitClient(connection)
    hub.clients = append(hub.clients, client)

    go hub.ListenClient(client)
}

/**
 * The function decides what to do for the client requests.
 * @param Client client
 */
func (hub *Hub) ListenClient(client *Client) {
    for {
        in := <-client.in
        if (util.IsIdentityCommand(in)) {
            hub.TellIdentity(client)
        } else if (util.IsListCommand(in)) {
            hub.ListClients(client)
        } else if (util.IsSendMessageCommand(in)) {
            hub.SendMessage(client, in)
        } else if (util.IsQuitCommand(in)) {
            hub.UnjoinClient(client)
        }
    }
}

/**
 * The function prints data to the hub.
 * @param string message
 */
func (hub *Hub) Write(message []byte) {
    for _, b := range message {
        hub.writer.WriteByte(b)
    }
    hub.writer.Flush()
}

/**
 * The function listens to the new connections and incoming data
 */
func (hub *Hub) ListenChannels() {
    for {
        select {
        case data := <-hub.in:
            hub.Write(data)
        case conn := <-hub.connections:
            hub.Join(conn)
        }
    }
}

/**
 * The function implements the /msg command.
 * @param Client fromClient
 * @param string message
 */
func (hub *Hub) SendMessage(fromClient *Client, message []byte) {
    if (bytes.Count(message, []byte(" ")) <= 1) {
        fromClient.out <- []byte("hub> Invalid /msg command parameters. Use /msg [user_id1,user_id2,...] [msg]\n")
        return
    }

    s := bytes.SplitN(message, []byte(" "), 3);
    receivers, body := s[1], s[2]
    r := bytes.Split(receivers, []byte(","))

    for _, client := range hub.clients {
        for _, receiver := range r {
            if (fmt.Sprintf("%d", client.user_id) == string(receiver)) {
                client.out <- append([]byte(fmt.Sprintf("%d", fromClient.user_id) + "> "), body...)
            }
        }
    }
}

/**
 * The function implements the /list command.
 * @param Client forClient
 */
func (hub *Hub) ListClients(forClient *Client) {
    onlyMe := true
    for _, client := range hub.clients {
        if (forClient.user_id != client.user_id) {
            forClient.out <- []byte("hub> " + fmt.Sprintf("%d", client.user_id) + "\n")
            onlyMe = false
        }
    }
    if (onlyMe == true) {
        forClient.out <- []byte("hub> No one else here :(\n")
    }
}

/**
 * The function implements the /quit command.
 * @param Client client
 */
func (hub *Hub) UnjoinClient(client *Client) {
    var tmpClients = make([]*Client, 0)
    for _, c := range hub.clients {
        if (c.user_id != client.user_id) {
            tmpClients = append(tmpClients, c)
        }
    }
    client.out <- []byte("/quit\n")
    hub.clients = tmpClients
}

/**
 * The function implements the /whoami command.
 * @param Client client
 */
func (hub *Hub) TellIdentity(client *Client) {
    client.out <- []byte("hub> " + fmt.Sprintf("%d", client.user_id) + "\n")
}

/**
 * The function initialize a hub.
 * @return Hub hub
 */
func InitHub() *Hub {
    writer := bufio.NewWriter(os.Stdout)

    hub := &Hub{
        clients: make([]*Client, 0),
        connections: make(chan net.Conn),
        in: make(chan []byte),
        out: make(chan []byte),
        writer: writer,
    }

    go hub.ListenChannels()

    return hub
}

/**
 * The function initialize a new client.
 * @return Client
 */
func InitClient(connection net.Conn) *Client {
    writer := bufio.NewWriter(connection)
    reader := bufio.NewReader(connection)

    // generate unique user_id for the client
    user_id := uint64(time.Now().UnixNano())

    client := &Client{
        in: make(chan []byte),
        out: make(chan []byte),
        reader: reader,
        writer: writer,
        user_id: user_id,
    }

    go client.Read()
    go client.Send()

    return client
}

func (hub *Hub) AttachConnection(connection net.Conn) {
    hub.connections <- connection
}
