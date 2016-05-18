The bothersome part of openconnect or anyconnect is that it hijacks all traffic on your computer. You can configure openconnect to use a user level program that can be wrapped in another program to selectively choose which traffic is sent. You can setup FoxyProxy to send just some traffic to that user level program with openconnect. However, these options are orders of magnitude slower than normal internet traffic.

For this solution, we start a container that sends all of its traffic to the configured proxy. There's a second container that sets up a user configurable proxy that will select which traffic to send across proxy and which traffic to access directly.


Start the connect containers:

	USER={username} PASS={password} docker-compose up

Configure your traffic to go through an HTTP proxy:

    HTTP_PROXY=http://localhost:8080
