import subprocess
import os
import math

def chunk_flac_ffmpeg(input_flac, output_prefix="chunk", chunk_length_sec=10.0):
    """
    Splits a FLAC file into smaller FLAC chunks using ffmpeg without re-encoding.

    Args:
      input_flac (str): Path to the input .flac file.
      output_prefix (str): Prefix to prepend to each chunk filename.
      chunk_length_sec (float): Length of each chunk in seconds.

    Returns:
      List of filenames of the chunked FLAC files.
    """
    # Ensure the input file exists
    if not os.path.exists(input_flac):
        raise FileNotFoundError(f"Input file {input_flac} not found.")

    # Get the total duration of the input FLAC file using ffprobe
    cmd_duration = [
        "ffprobe", 
        "-v", "error", 
        "-show_entries", "format=duration", 
        "-of", "default=noprint_wrappers=1:nokey=1", 
        input_flac
    ]
    try:
        duration = float(subprocess.check_output(cmd_duration).strip())
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to get duration: {e}")

    # Calculate the number of chunks
    num_chunks = math.ceil(duration / chunk_length_sec)

    output_files = []

    for i in range(num_chunks):
        start_time = i * chunk_length_sec
        out_fname = f"{output_prefix}_{i+1:03d}.flac"
        
        # Use ffmpeg to extract the chunk
        cmd_chunk = [
            "ffmpeg",
            "-i", input_flac,
            "-ss", str(start_time),
            "-t", str(chunk_length_sec),
            "-c", "copy",  # Copy codec to avoid re-encoding
            out_fname
        ]

        try:
            subprocess.run(cmd_chunk, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            output_files.append(out_fname)
            print(f"Created chunk {out_fname} starting at {start_time}s.")
        except subprocess.CalledProcessError as e:
            print(f"Failed to create chunk {out_fname}: {e}")

    return output_files

if __name__ == "__main__":
    input_file = "The Weeknd - Timeless.flac"
    chunks = chunk_flac_ffmpeg(input_file, output_prefix="my_chunk", chunk_length_sec=10.0)
    print("Done splitting. Chunks:", chunks)