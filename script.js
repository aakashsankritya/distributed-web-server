const axios = require('axios');

// Configuration
const endpoint = 'http://localhost:8089/api/verve/accept'; // Replace with your server's URL
const idCount = 100; // Number of unique IDs to use
const parallelRequests = 500; // Number of concurrent requests
const requestInterval = 10; // Time between each batch of parallel requests in milliseconds

// Generate a fixed set of random IDs
const ids = Array.from({ length: idCount }, () => `${Math.floor(Math.random() * 1000000)}`);

// Function to send a GET request for a specific ID
async function sendRequest(id) {
    try {
        const response = await axios.get(endpoint, { params: { id } });
        console.log(`Request sent for id: ${id}, Response: ${response.data}`);
    } catch (error) {
        console.error(`Error sending request for id: ${id}`, error.message);
    }
}

// Function to start parallel requests
async function sendParallelRequests() {
    while (true) {
        console.log('Starting a batch of parallel requests...');

        const promises = [];
        for (let i = 0; i < parallelRequests; i++) {
            // Pick a random ID from the list
            const id = ids[Math.floor(Math.random() * ids.length)];
            promises.push(sendRequest(id));
        }

        // Wait for all requests in this batch to complete
        await Promise.all(promises);

        // Wait before sending the next batch
        await new Promise(resolve => setTimeout(resolve, requestInterval));
    }
}

// Start the parallel request loop
console.log(`Generated IDs: ${ids.join(', ')}`);
console.log('Starting to send parallel requests...');
sendParallelRequests();
