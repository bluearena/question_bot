## Question bot

[![Go Report Card](https://goreportcard.com/badge/github.com/halink0803/question_bot)](https://goreportcard.com/report/github.com/halink0803/question_bot)

## Build 
Before build bot, remember to prepare 2 files: config.json and questions.json following the sample files in the repository.

### Docker

```
    <!-- docker build -t question . -->
    docker-compose build
```

## Run

### Docker

tee for logging purpose
```
    <!-- docker run question question_bot | tee log.log -->
    docker-compose up
```