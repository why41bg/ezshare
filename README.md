EZShare is a web-based real-time screen sharing tool that allows you to share your screen with your friends without logging in! EZShare transmits video data via STUN/TURN, but will never collect your data, so you can use it with confidence.

- [x] TURN/STUN server
- [x] Session store users information
- [x] WebSocket signaling pipeline 
- [x] video streaming
- [ ] audio streaming

# How to deploy

Clone this repository to your local location. Then follow the steps below.

1. You first need to apply for a CA certificate to enable HTTPS. Because WebRTC requires HTTPS to work.
2. Create `ezshare.config.development` or `ezshare.config` locally as the configuration file, the former has higher priority.
3. Modify the relevant configurations in the `vite.config.mts` file under the ui folder.

``` bash
# Install go related dependencies
go mod tidy

# Go to the root directory of ezshare and start the backend
go run main.go start

# Switch to the ui directory and install related dependencies
yarn install

# Start the front end separately
yarn start
```
