
/**
 * The slider bar input element used for seeking through the audio track.
 * @type {HTMLAudioElement}
 */
const audioPlayer = document.getElementById('audio-player');
/**
 * The slider bar input element used for seeking through the audio track.
 * @type {HTMLInputElement}
 */
const seekBar = document.getElementById('seek-bar');
const playBtn = document.getElementById('play-btn');
const pauseBtn = document.getElementById('pause-btn');

const loadBtn = document.getElementById('load-btn');

// Variables for MediaSource and SourceBuffer
let mediaSource;
let sourceBuffer;
let audioChunks = {};
let appendedChunks = new Set();

// Variables for audio file and metadata
let fileSize;
let metaData;


// Event listener for loading the song
loadBtn.addEventListener('click', async () => {
    // Prompt user for the audio file URL and metadata URL
    const audioURL = 'http://127.0.0.1:3000/streamcached?id=z1gyab4962a8rj6';
    const metadataURL = 'http://127.0.0.1:3000/metadata?id=z1gyab4962a8rj6';

    if (!audioURL || !metadataURL) {
        alert('Please enter valid audio and metadata URLs');
        return;
    }
       
    // Fetch metadata JSON
    const response = await fetch(metadataURL);
    const metadata = await response.json();
    metaData = metadata;
    keys = Object.keys(metadata["chunks"])
    //divide the timeline into equally seperated keys
    fileSize = metadata["file_size"]

    seekBar.max = fileSize;


    // Initialize MediaSource
    mediaSource = new MediaSource();
    // Set the audio source to the MediaSource object
    audioPlayer.src = URL.createObjectURL(mediaSource);
    // Event listener when MediaSource is open
    mediaSource.addEventListener('sourceopen', () => initSourceBuffer(metadata, audioURL));
});

// Initialize SourceBuffer and preload initial chunks
async function initSourceBuffer(metadata, audioURL) {
    
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
});

// Seek to new time when seek bar value changes
seekBar.addEventListener('input', () => {
    audioPlayer.currentTime = seekBar.value;
});

//when timeline updats, check the current byte. if the chunk for the current byte is loaded then let the audioplayer play.
//when the current byte is close to the end of the loaded chunk, we can request chunk from endpoint.
//the request chunk endpoint is dynamic, meaning we can request multiple chunks in single call.