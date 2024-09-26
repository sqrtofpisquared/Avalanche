# Stream Principles

In avalanche, data streams are not for the purpose of transmitting data from one point to another, but rather for
synchronizing the _state_ of a system across multiple nodes. This is a subtle but important distinction. This document
will go into detail about the principles that underlie the avalanche streaming.

## Stream State and Data Source

Every avalanche stream is tied to a data source. This could be something like a screen, audio source, sensor, or any
other source of data that has an active state at any given point in time. The stream is responsible for delivering the
details of this state to another node in the network, allowing for "state replication" or an approximation thereof
remotely.

For example, take a common use case of a user remotely controlling a computer. They need to receive the current state:

- The display
- The audio output

They also need to send the following states to the computer:

- Keyboard
- Mouse
- Microphone
- Webcam

The stream is responsible for delivering the _current state_ of these data sources to the remote node. This does have
some implications for the approach to data source. Specifically in regard to cumulative data sources, such as mouse
input. The stream must be able to deliver the _current_ state of the mouse, not just the delta from the last state, and
as such, the client would need to keep track of deltas internally. In the specific case of mouse movement, this would
involve transmitting a new mouse position every time the mouse moves, and on the receiving end, interpolating the mouse
movement to account for missing data.

For very high temporal resolution data sources, such as audio, the data must be sent in chunks - we can't send 44,100
packets per second, so instead the current state is sent in chunks that represent a small window of time. While the
literal transmission of the data is chunked, the stream is still considered to transmit the _current state_ of the
audio source, as opposed to transmitting the audio data in a more traditional sense.

## Stream encoding and decoding

Every stream must have an encoder and a decoder. The encoder is responsible for taking the current state of the data and
transforming it into a format that can be transmitted over the network. The decoder is responsible for taking the
transmitted data and transforming it back into the current state of the data. Encoders and decoders can be as simple as
just serializing a couple integers representing a mouse position, or as complex as an AV1 encoder for video data.

Encoders are responsible for:

- Taking in a standardized state format for a given data type
- Serializing the data into a format that can be transmitted over the network
- Packaging the data into chunks to be transmitted over the network
- Ensuring that the data is transmitted in a timely manner
- Adjusting the data rate to match what the client desires (e.g. lower bitrate for a slower connection)

Decoders are responsible:

- Taking in the serialized data
- Reconstructing the data into the standardized state format
- Ensuring that the data is reconstructed in a timely manner

Encoders and decoders are generally paired - a given encoder type will support only specific decoder types, and vice
versa.

## No chunk is more important than another

A key principle of avalanche streams is that the no chunk is more important than another. For all streams of all types,
losing a single packet should minimally degrade the quality of the stream. While there can be some threshold of packet
loss at which a stream becomes unusable, the quality should degrade gracefully as packets are lost.

This is particularly important for encoding methods that rely on temporal encoding, such as video encoding. Crucially:

- If bandwidth allows, opt for non-temporal methods to avoid temporal dependencies entirely.
- If temporal encoding is a must for bandwidth reasons, reference data should not be sent separate from deltas. Opt for
  intra-refresh or similar techniques to avoid this. When using intra-refresh, the reference data should be sent
  alongside the deltas in any given packet.