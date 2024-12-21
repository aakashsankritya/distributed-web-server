const axios = require('axios');

const endpoint = 'http://localhost:8089/api/verve/accept';
const nginxEndpoint = 'http://nginx-dev:8089/api/verve/accept';

const idCount = 10000; // Total number of unique IDs
const parallelRequests = 10000; // Number of requests per batch (10K requests per sec / 10)
// const requestsPerSecond = 10000; // Target 10K requests per second
const batchInterval = 1000

let successCount = 0;
let failureCount = 0;

const ids = Array.from({ length: idCount }, () => `${Math.floor(Math.random() * 1000000)}`);

async function sendRequest(id, passEndpoint = false) {
    try {
        const params = { id };
        if (passEndpoint) {
            params.endpoint = endpoint
        }
        const response = await axios.get(endpoint, { params: params });
        if (response.status === 200) {
            successCount++;
            console.log(`Request sent for id: ${id}, Response: ${response.data}`);
        } else {
            failureCount++;
            console.error(`Request failed for id: ${id}, Response status: ${response.status}`);
        }
    } catch (error) {
        failureCount++;
        console.error(`Error sending request for id: ${id}`, error.message);
    }
}

async function sendParallelRequests() {
    while (true) {
        console.log('Starting a batch of parallel requests...');

        const promises = [];
        for (let i = 0; i < parallelRequests; i++) {
            // Pick a random ID from the list
            const id = ids[Math.floor(Math.random() * ids.length)];
            promises.push(sendRequest(id));
        }

        // for (let i = 0; i < parallelRequests * 0.10; i++) {
        //     // Pick a random ID from the list
        //     const id = ids[Math.floor(Math.random() * ids.length)];
        //     promises.push(sendRequest(id, true));
        // }
        // Wait for all requests in this batch to complete
        await Promise.all(promises);
        console.log(`Requests processed: ${successCount + failureCount}, Successes: ${successCount}, Failures: ${failureCount}`);
        // Wait to maintain the desired requests per second
        await new Promise(resolve => setTimeout(resolve, batchInterval));
    }
}

console.log(`Generated IDs: ${ids.join(', ')}`);
console.log('Starting to send parallel requests...');
sendParallelRequests();
