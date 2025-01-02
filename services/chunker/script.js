// Get references to DOM elements
const audioPlayer = document.getElementById('audio-player');
const playBtn = document.getElementById('play-btn');
const pauseBtn = document.getElementById('pause-btn');
const seekBar = document.getElementById('seek-bar');
const loadBtn = document.getElementById('load-btn');

// Variables for MediaSource and SourceBuffer
let mediaSource;
let sourceBuffer;
let audioChunks = {};
let appendedChunks = new Set();

// Event listener for loading the song
loadBtn.addEventListener('click', async () => {
    // Prompt user for the audio file URL and metadata URL
    const audioURL = prompt('Enter the URL of the audio file:');
    const metadataURL = prompt('Enter the URL of the metadata JSON file:');

    if (audioURL && metadataURL) {
        // Fetch metadata JSON
        const response = await fetch(metadataURL);
        const metadata = await response.json();

        // Initialize MediaSource
        mediaSource = new MediaSource();

        // Set the audio source to the MediaSource object
        audioPlayer.src = URL.createObjectURL(mediaSource);

        // Event listener when MediaSource is open
        mediaSource.addEventListener('sourceopen', () => initSourceBuffer(metadata, audioURL));
    }
});

// Initialize SourceBuffer and preload initial chunks
async function initSourceBuffer(metadata, audioURL) {
    // Add SourceBuffer for AAC audio or adjust codec as needed
    sourceBuffer = mediaSource.addSourceBuffer('audio/flac');

    // Preload initial chunks specified in metadata
    await preloadChunks(metadata.initialChunks, audioURL);

    // Append preloaded chunks to SourceBuffer
    for (const chunk of metadata.initialChunks) {
        if (audioChunks[chunk.id]) {
            sourceBuffer.appendBuffer(audioChunks[chunk.id]);
            appendedChunks.add(chunk.id);
        }
    }
}

// Function to preload chunks
async function preloadChunks(chunksInfo, audioURL) {
    for (const chunk of chunksInfo) {
        // Fetch chunk using byte range
        const response = await fetch(audioURL, {
            headers: {
                Range: `bytes=${chunk.start}-${chunk.end}`,
            },
        });
        const arrayBuffer = await response.arrayBuffer();
        audioChunks[chunk.id] = arrayBuffer;
    }
}

// Play button event listener
playBtn.addEventListener('click', () => {
    audioPlayer.play();
});

// Pause button event listener
pauseBtn.addEventListener('click', () => {
    audioPlayer.pause();
});

// Update seek bar as audio plays
audioPlayer.addEventListener('timeupdate', () => {
    // Get the current time and duration
    const currentTime = audioPlayer.currentTime;
    const duration = audioPlayer.duration || 0;

    // Update the seek bar value
    seekBar.max = duration;
    seekBar.value = currentTime;

    // Check if we need to append new chunks
    checkAndAppendChunks(currentTime);
});

// Seek to new time when seek bar value changes
seekBar.addEventListener('input', () => {
    audioPlayer.currentTime = seekBar.value;
});

// Function to check and append chunks based on current time
async function checkAndAppendChunks(currentTime) {
    // Determine which chunk is needed based on current time
    // For simplicity, assuming each chunk is 5 seconds long
    const chunkDuration = 5; // adjust as per actual chunk duration
    const chunkId = Math.floor(currentTime / chunkDuration);

    // If chunk is not appended yet
    if (!appendedChunks.has(chunkId)) {
        // Fetch chunk metadata
        const chunkInfo = await getChunkInfo(chunkId);
        if (chunkInfo) {
            // Preload chunk if not in memory
            if (!audioChunks[chunkId]) {
                await preloadChunks([chunkInfo], chunkInfo.audioURL);
            }
            // Append chunk to SourceBuffer
            sourceBuffer.appendBuffer(audioChunks[chunkId]);
            appendedChunks.add(chunkId);
        }
    }
}

// Function to get chunk info (simulate metadata retrieval)
async function getChunkInfo(chunkId) {
    // Implement logic to get chunk metadata based on chunkId
    // For simulation, we'll generate chunk info
    const audioURL = audioPlayer.src; // Assuming same audio URL
    const chunkDuration = 5; // adjust as per actual chunk duration
    const startByte = chunkId * chunkDuration * 16000; // assuming bitrate of 128kbps
    const endByte = startByte + (chunkDuration * 16000) - 1;

    return {
        id: chunkId,
        start: startByte,
        end: endByte,
        audioURL: audioURL,
    };
}