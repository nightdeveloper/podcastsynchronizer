set GOPATH=%CD%

call go install github.com/nightdeveloper/podcastsynchronizer/main

call "bin/main.exe"
