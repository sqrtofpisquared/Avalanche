# Avalanche
A no-compromises multi-client real-time streaming system built for and by enthusiasts.

Avalanche is designed for enthusiasts to control and use many computers or devices at once over a network without compromising on quality or latency.

This system is designed for high-end home networks. Avalanche may not be right for you if you have a high congestion, primarily wireless, or long distance network.

Avalanche optimizes for minimum latency, maximum quality, and broad device support. It doesn't optimize for security or packet loss compensation.


# Key Concepts

## Client
A client is a device that is connected to an Avalanche network. Clients can transmit and receive streams, along with provide data sources.

## Stream
A stream is a monodirectional, realtime transmission of data. Streams are time-sensitive, and are not designed to ensure delivery. 
Streams transmit timestamps for the purpose of synchronization.
Streams are separate from their data source, so if a stream disconnects, the underlying data source for that stream would not care.
Streams are provided by a stream serializer, and received by a stream deserializer.

## Data source
The source of data for a stream, such as a desktop, game, audio source, camera, human input device, etc.

## Stream serializer/deserializer
A system to prepare data for transmission, and use transmitted data. These systems can get quite complicated, and will handle things like compression, packaging, etc. 
These may tie into hardware for acceleration.

## Stream post-processors
These are clientside systems that can augment recieved data to do things like upscaling, deblocking, stream prediction, etc.

## Client network
A peer-to-peer management network for all clients. It allows clients to establish streams between each other, and can help clients prioritize streams to optimize user experience.
