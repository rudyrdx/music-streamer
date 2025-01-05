import os

def read_flac_headers(flac_file_path):
    
    with open(flac_file_path, 'rb') as f:
        # Read the "fLaC" marker
        signature = f.read(4)
        if signature != b'fLaC':
            raise ValueError("Not a valid FLAC file.")

        headers = signature
        is_last_metadata_block = False

        # Read metadata blocks until the last one
        while not is_last_metadata_block:
            # Each metadata block begins with a 1-byte header
            header_byte = f.read(1)
            if len(header_byte) == 0:
                raise ValueError("Unexpected end of file while reading metadata headers.")

            header_int = header_byte[0]
            is_last_metadata_block = (header_int & 0x80) != 0  # Check if the last bit is set
            block_type = header_int & 0x7F  # Lower 7 bits
            # Next 3 bytes are the length of the metadata block
            length_bytes = f.read(3)
            if len(length_bytes) < 3:
                raise ValueError("Unexpected end of file while reading metadata block length.")
            block_length = int.from_bytes(length_bytes, byteorder='big')

            # Read the block data
            block_data = f.read(block_length)
            if len(block_data) < block_length:
                raise ValueError("Unexpected end of file while reading metadata block data.")

            # Append the metadata block to headers
            headers += header_byte + length_bytes + block_data

        # At this point, headers contain all metadata blocks
        return headers

def find_frame_positions(data):
    frame_positions = []
    data_len = len(data)
    i = 0
    while i < data_len - 1:
        byte1 = data[i]
        byte2 = data[i+1]
        word = (byte1 << 8) | byte2
        # The FLAC frame sync code is 14 bits: 0x3FFE
        sync = (word >> 2) & 0x3FFF  # Get top 14 bits
        if sync == 0x3FFE:
            # Found a frame sync code
            frame_positions.append(i)
            i += 2  # Move past the sync code
        else:
            i += 1
    return frame_positions

def split_flac_file_at_frames(flac_file_path, output_dir, chunk_size=1024*1024):
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    headers = read_flac_headers(flac_file_path)

    with open(flac_file_path, 'rb') as f:
        # Skip the headers we have already read
        f.seek(len(headers))
        audio_data = f.read()  # Read the rest of the file

    frame_positions = find_frame_positions(audio_data)
    if not frame_positions:
        raise ValueError("No FLAC frames found in the audio data.")
    print(f"Found {len(frame_positions)} frame positions.")

    current_frame_index = 0
    chunk_num = 1

    while current_frame_index < len(frame_positions):
        accumulated_size = 0
        start_frame_index = current_frame_index

        while accumulated_size < chunk_size and current_frame_index < len(frame_positions) - 1:
            frame_start = frame_positions[current_frame_index]
            frame_end = frame_positions[current_frame_index + 1]
            frame_length = frame_end - frame_start
            accumulated_size += frame_length
            current_frame_index += 1

        # Handle the last frame
        if current_frame_index == len(frame_positions) - 1:
            frame_start = frame_positions[current_frame_index]
            frame_end = len(audio_data)
            frame_length = frame_end - frame_start
            accumulated_size += frame_length
            current_frame_index += 1

        # Extract frames from start_frame_index to current_frame_index
        data_start = frame_positions[start_frame_index]
        if current_frame_index < len(frame_positions):
            data_end = frame_positions[current_frame_index]
        else:
            data_end = len(audio_data)
        chunk_data = audio_data[data_start:data_end]

        chunk_file_name = os.path.join(output_dir, f"chunk_{chunk_num:04d}.flac")
        with open(chunk_file_name, 'wb') as chunk_file:
            # Write the headers
            chunk_file.write(headers)
            # Write the chunk data
            chunk_file.write(chunk_data)

        print(f"Created {chunk_file_name} ({len(chunk_data)} bytes)")
        chunk_num += 1

    print("Splitting complete.")

# Example usage:
if __name__ == "__main__":
    flac_file_path = "tmp/4690724c-1a8c-4e7f-9df3-b84dbb4e4a50.flac"  # Path to your input FLAC file
    output_dir = "flac_chunks"     # Output directory to store the chunks
    chunk_size = 1024 * 1024       # 1MB chunks

    split_flac_file_at_frames(flac_file_path, output_dir, chunk_size)