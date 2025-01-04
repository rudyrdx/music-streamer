
const baseUrl = 'http://localhost:3000/';
const audio = new Audio();
const mediaSource = new MediaSource();
audio.src = URL.createObjectURL(mediaSource);

mediaSource.addEventListener('sourceopen', async () => {
  const srcBuff = mediaSource.addSourceBuffer('audio/webm; codecs="opus"');

  srcBuff.addEventListener('error', (e) => {
    console.error('SourceBuffer error:', e);
  });

  let metadata = null;
  let current_id = 1;

  try {
    const metaResponse = await fetch(baseUrl + 'metadata?id=z1gyab4962a8rj6');
    metadata = await metaResponse.json();
    console.log(metadata);
  } catch (error) {
    console.error("Failed to fetch metadata:", error);
    return;
  }

  if (!metadata) return;

  let id = metadata["chunks"][current_id.toString()]["id"];
  if (id === undefined) return;

  try {
    const initialChunk = await fetchChunk(id);
    srcBuff.appendBuffer(initialChunk);
    audio.play();
  } catch (error) {
    console.error("Failed to fetch or append initial chunk:", error);
    return;
  }

  srcBuff.addEventListener('updateend', async () => {
    try {
      const nextChunk = await fetchChunk(id);
      if (srcBuff.readyState === 'open') {
        srcBuff.appendBuffer(nextChunk);
      }
    } catch (error) {
      console.error("Failed to fetch or append next chunk:", error);
    }

    if (audio.currentTime > srcBuff.buffered.end(0) - 3) {
      current_id++;
      id = metadata["chunks"][current_id.toString()]["id"];
      if (id === undefined) return;

      try {
        const subsequentChunk = await fetchChunk(id);
        if (srcBuff.readyState === 'open') {
          srcBuff.appendBuffer(subsequentChunk);
        }
      } catch (error) {
        console.error("Failed to fetch or append subsequent chunk:", error);
      }
    }
  });
});

async function fetchChunk(chnkId) {
  const response = await fetch(baseUrl + 'chunk?id=' + chnkId);
  const data = await response.arrayBuffer();
  return data;
}
