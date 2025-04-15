build:
	go build -o blockchain_app

start:
	./blockchain_app --server -process

frontend:
	cd ../blockchain-visualizer && npm install && npm run dev

all: build start frontend
