import asyncio
import aiohttp

async def fetch_chunk(session, url):
    async with session.get(url) as response:
        if response.status == 206:
            r = await response.read()
            print(f"Fetched {url}: {r[:100]}")
        else:
            print(f"Error fetching {url}: {response.status}")
            return None

async def main():
    base_url = "http://127.0.0.1:3000/chunk?id="
    chunk_ids = ["m5rpfmr8j3gutv6"]  # Add your chunk IDs here
    
    async with aiohttp.ClientSession() as session:
        tasks = [fetch_chunk(session, f"{base_url}{chunk_id}") for chunk_id in chunk_ids]
        await asyncio.gather(*tasks)

if __name__ == "__main__":
    asyncio.run(main())