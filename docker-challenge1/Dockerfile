FROM golang:1.16-alpine AS build

WORKDIR /usr/src/app

COPY . /usr/src/app/
RUN go build -o /bin/project

FROM scratch
COPY --from=build /bin/project /bin/project
ENTRYPOINT ["/bin/project"]
CMD ["go", "run", "."]