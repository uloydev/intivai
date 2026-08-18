FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl make g++ python3 python3-pip
RUN curl -L https://github.com/ggerganov/whisper.cpp/archive/refs/tags/v1.7.0.tar.gz | tar xz
WORKDIR /whisper.cpp-1.7.0
# Build the server executable
RUN make -j server
# Download the tiny model for dev/testing
RUN bash ./models/download-ggml-model.sh tiny
EXPOSE 8080
# Run the OpenAI compatible server
ENTRYPOINT ["./server", "-m", "models/ggml-tiny.bin", "--host", "0.0.0.0", "--port", "8080"]
