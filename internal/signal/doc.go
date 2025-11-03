// Package signal provides a client for interacting with Signal Messenger via signal-cli.
//
// This package offers Signal messaging functionality including:
//   - Sending direct messages to Signal users
//   - Sending messages to Signal groups
//   - Receiving messages with timeout support
//   - Group management and lookup
//
// The client wraps the signal-cli command-line tool and requires signal-cli to be
// installed and configured on the system. Users must register their Signal account
// with signal-cli before using this package.
//
// Prerequisites:
//  1. Install signal-cli: https://github.com/AsamK/signal-cli
//  2. Register your Signal account:
//     signal-cli -u YOUR_PHONE_NUMBER register
//  3. Verify your account with the SMS code:
//     signal-cli -u YOUR_PHONE_NUMBER verify CODE_RECEIVED
//
// Authentication:
// Signal credentials are managed by signal-cli and stored in the signal-cli data directory
// (typically ~/.local/share/signal-cli/). Each client instance is associated with a
// specific phone number.
//
// Example usage:
//
//	ctx := context.Background()
//	client, err := signal.NewClient(ctx, "+15551234567")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send a direct message
//	err = client.SendMessage("+15559876543", "Hello from Signal!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send a group message
//	err = client.SendGroupMessage("Family Chat", "Hello everyone!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Receive messages with a 30 second timeout
//	response, err := client.ReceiveMessage(30)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if response.Message != "" {
//	    fmt.Printf("Received: %s from %s\n", response.Message, response.SenderID)
//	}
package signal
