# Coding Assignment Submission and Discussion

## Deliverables

The program was run as per the instructions in this [README](../README.md) file. The results can be found [here](results.txt).

## Solution overview

The problem statement was straightforward: implementing a REST program/service, in Go to collect metrics reported by devices or clients running at the edge, and can be used to query statistics related to these. The openapi specification for the interface was provided.

No external libraries were used, but the implementation requires Go 1.22+ for pattern matching in the net/http package.

The statistics in the requirements only included :
- uptime (defined as the percentage of heartbearts received in relation to the expected number during the time period considered)
- average of video upload times

For this exercise, there were no additional non-functional requirements related to persistence, scalability and performance e.g. rate of query/updates and number of devices to consider.

Based on the metrics to be collected per device, the natural choice was to use an in-memory store (hash map) to keep track of each devices statistics. Per the requirements, the statistics only required a fixed size structure for each device. 
- The uptime only requires to keep track of the first, last and number of heartbeats received.
- The average upload time the total number of nanoseconds and the number of samples.

As such, the time complexity for all operations is constant O(1), since all accesses use a hash map lookup, and only operate on a fixed size record, and the space complexity is linear O(N), proportional to the number of devices.

The implementation was structured as a classic service, comprised of an api/handlers (controller) to handle the incoming REST requests, service (business logic to handle the stats reported by the devices and compute the statistics), and data (in-memory storage).

The configuration options were kept to a minimum, controlled with environment variables or command-line options (config module).

The list of devices to monitor is preloaded from a csv file, and remains static for the duration of the execution (could be a further improvement to make it dynamic, but not called for in the challenge requirements.) This is implemented in module device.


## Implementation Strategy

Since I am new to Go, and this was a relatively simple and self-contained project, I decided to make extensive use of AI coding tools, mostly Claude Code. This was both as an experiment, and also as a way to learn the idioms in Go (eg how to structure a standard project, write unit tests, make good use of the available libs, and properly use common patterns). This proved successful.

I first used to the desktop application to present the problem, explain the requirements as I understood them and with my ideas on how to solve the problem and structure the solution at a high level. As such, I chose NOT to copy the instructions provided werbatim. After a few iterations, the AI and I converged on a plan, which resulted on a [prompt](./claude-code-prompt.md) that I could feed to Claude code (CLI).

The analysis of the problem and interaction with Claude desktop took about 30 min. The initial implementation took 5-6 min of "thinking" by the model. There were mistakes, and between the rework done by the model and manually, I spent an overall time of 2 to 2.5 hours to get a working implementation (including the time to become familiar withthe go toolchain), and then some more time to review and inspect the generated code.

Note: For authenticity, this document was *NOT* generated with AI :)


## Topics for further extension

This section lists a few ideas to address limitations of the current implementation, and what would be needed to make it more production ready.

### Uptime implementation

Currently, the uptime computation is based on an assumption that the heart arrive in order, and are reported on a fixed schedule (60s). A more robust implementation could be to keep track of a number of heartbearts (eg 10-20, rotating out the older ones) and consider that a heartbeat was received for a minute (or arbitratry time period, eg 30s, 60s, 120s) if at least one heartbeat was received for that period.

It would make the calculation more accurate if a device sends them more often.

Also now the uptime is calculated since teh startup ofthe metrics system. It could be useful, or more meaningful to report it for the last e.g. 15m, 30m, hour, day, etc

### Video upload times

Similarly to the uptime, the current requirements call for an average of upload times since the system was started, and as such the implementation only keeps track of the total times and number of reported events. As the system keeps running, the impact of outliers will be less visible and the statistic may lose some usefulness. 

There could be a periodic reset, e.g. to reflect the avg over the last e.g. 15m, 30m, hour, day, etc.

### Extend metrics set

It would be natural to expand the set of metrics collected for a given device. Example I can think of could be the last reboot time, number of reboots, video duration, size, etc. We could also keep track of other aggregates values like the min and max in addition to the average.

In this case, if a number of such statistics is anticipated, it could be better to enhance the model to store a more dynamic list of metrics for each device, instead of adding more fields to the device record for each new statistic. Since the values to report (eg min, max, avg, count, etc) are similar for one statistics to the next, we should think of a generic structure that can be used for any metric. Since this problem is well understood, inpiration could be taken from systems like Prometheus, and even add more interesting features like histograms.


### Deployment, scaling and state sharing

Currently, the implementation stores the metrics in memory, and there is only one instance of the program running. In order to scale the solution, we would need to be able to deploy multiple instances, and use a database to share the state across them (i.e. metrics for the devices).

In this case we could easily containerize the application, deploy on Kubernetes, or use a cloud provided service (e.g. AWS ECS, or its equivalent on other platforms + API Gateway) and use an in-memory DB (eg Redis, MemCached or alternatives) to share the state.


### Authetication/authorization and security

To take this to production we would need to add authentication. At the minimum use mTLS to secure communication between the devices and the service, and potentially use JWT for finer grained authorization, for example if the devices need to communicate with different services for different purposes.

The implementation should also be reviewed to protect against vulnerabilities, for example to protect against large message, and  make sure all input is validated.

In terms of deployment, the necessary measures only API gateway, vs Web Application Firewall) would depend where devices are deployed, if they communicate directly with the service, and how the service is deployed.


### Persistence and timeseries

The current implementation does not provide persistence, or historical data. The move to a time-series database could solve this problem elegantly and provide a richer set of queries (eg the average of a metric over the last 15m, for th last week). This would also allow the use of tools like Grafana to nicely visualize these.

### Dynamic management of devices

For this project, the list of devices was statically defined in a CSV file and remained the same for the duration of the execution. This limitation would need to be lifted, a mechanism for devices to dynamically register could be implemented.
