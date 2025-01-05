// Define variables outside to make them accessible in functions
const baseUrl = 'http://localhost:3000/';

/**
 * @type {MediaSource}
 */
let mediaSource = null;

/**
 * @type {HTMLAudioElement}
 */
const audio = document.getElementById('audioElement');

/**
 * @type {JSON}
 */
let metadata = null;

/**
 * @type {SourceBuffer}
 */
let srcBuff = null;

let current_id = 1;
let pendingChunkIds = new Set();

function updateStatus(message) {
    document.getElementById('status').innerText = 'Status: ' + message;
}

function loadMusic() {
    if (mediaSource) {
        // If mediaSource already exists, reset everything
        mediaSource.removeEventListener('sourceopen', sourceOpen);
        mediaSource = null;
        audio.src = '';
        current_id = 1;
        pendingChunkIds.clear();
        if (srcBuff) {
            srcBuff.abort();
            srcBuff = null;
        }
    }
    
    mediaSource = new MediaSource();
    audio.src = URL.createObjectURL(mediaSource);
    mediaSource.addEventListener('sourceopen', sourceOpen);
    updateStatus('Loading music...');
}

async function sourceOpen() {
    try {
        const metaResponse = await fetch(baseUrl + 'metadata?id=z1gyab4962a8rj6');
        metadata = await metaResponse.json();
        console.log(metadata);
        updateStatus('Metadata fetched.');
    } catch (error) {
        console.error("Failed to fetch metadata:", error);
        updateStatus('Failed to fetch metadata.');
        return;
    }

    srcBuff = mediaSource.addSourceBuffer('audio/webm; codecs="opus"');
    if (!metadata) return;

    current_id = 1;
    pendingChunkIds = new Set();

    const initial_id = metadata["chunks"][current_id.toString()]["id"];
    if (initial_id === undefined) return;
    
    pendingChunkIds.add(current_id);
    await fetchAndAppendChunk(initial_id, srcBuff);
    pendingChunkIds.delete(current_id);
    updateStatus('Playing music...');
    srcBuff.addEventListener('updateend', updateEnd);
    // audio.addEventListener('timeupdate', timeUpdateHandler);
}

async function updateEnd() {
   console.log('Update end');
}
audio.addEventListener('canplay', () => {
    audio.play();
});


async function timeUpdateHandler() {
    if (srcBuff.buffered.length > 0 &&
        audio.currentTime > srcBuff.buffered.end(0) - 3 &&
        !pendingChunkIds.has(current_id + 1)) {

        const next_id = current_id + 1;
        const next_chunk_info = metadata["chunks"][next_id.toString()];

        if (!next_chunk_info) {
            updateStatus('End of music.');
            return;
        }

        const next_chunk_id = next_chunk_info["id"];
        pendingChunkIds.add(next_id);

        // Await fetchAndAppendChunk, but don't play here
        await fetchAndAppendChunk(next_chunk_id, srcBuff);

        pendingChunkIds.delete(next_id);
        current_id = next_id;
    }
}
function fetchNextChunk() {
    const next_id = current_id + 1;
    if (pendingChunkIds.has(next_id)) {
        console.log("Chunk is already being fetched");
        return;
    }
    const next_chunk_info = metadata["chunks"][next_id.toString()];
    if (!next_chunk_info) {
        updateStatus('No more chunks to fetch.');
        return;
    }

    const next_chunk_id = next_chunk_info["id"];
    pendingChunkIds.add(next_id);

    fetchAndAppendChunk(next_chunk_id, srcBuff).then(() => {
        pendingChunkIds.delete(next_id);
        current_id = next_id;
    });
}

/**
 * 
 * @param {number} chunkId 
 * @param {SourceBuffer} srcBuff 
 */
async function fetchAndAppendChunk(chunkId, srcBuff) {
    try {
        const chunkData = await fetchChunk(chunkId);
        await new Promise((resolve, reject) => {
            srcBuff.addEventListener('updateend', function onUpdateEnd() {
                srcBuff.removeEventListener('updateend', onUpdateEnd);
                resolve();
            });
            srcBuff.addEventListener('error', function onError(e) {
                srcBuff.removeEventListener('error', onError);
                reject(e);
            });
            srcBuff.appendBuffer(chunkData);
        });
        console.log('Appended chunk:', chunkId);
        updateStatus('Fetched and appended chunk: ' + chunkId);
    } catch (error) {
        console.error("Failed to fetch or append chunk:", error);
        updateStatus('Failed to fetch or append chunk.');
    }
}

async function fetchChunk(chnkId) {
    const response = await fetch(baseUrl + 'chunk?id=' + chnkId);
    if (!response.ok) {
        throw new Error('Network response was not ok');
    }
    const data = await response.arrayBuffer();
    console.log('Fetched chunk:', chnkId);
    return data;
}

// Wire the controls to functions
document.getElementById('loadMusicBtn').addEventListener('click', loadMusic);
document.getElementById('playBtn').addEventListener('click', () => { 
    audio.play();
    updateStatus('Playing music...');
});
document.getElementById('pauseBtn').addEventListener('click', () => { 
    audio.pause();
    updateStatus('Music paused.');
});
document.getElementById('stopBtn').addEventListener('click', () => { 
    audio.pause(); 
    audio.currentTime = 0;
    updateStatus('Music stopped.');
});
document.getElementById('nextChunkBtn').addEventListener('click', fetchNextChunk);