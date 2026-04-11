# Coding Assignment Submission and Discussion

## Deliverables

The program was run as per the instructions in this [README](../README.md) file. The results can be found [here](results.txt).

## Solution overview

The problem statement was straightforward: implementing a REST program/service in Go to collect metrics reported by devices or clients running at the edge, and can be used to query statistics related to these. 

No external libraries were needed, but the implementation requires Go 1.22+ for pattern matching in the net/http library.

The requirements only included :
- uptime (defined as the percentage of heartbearts received in relation to the expected number during the time period considered)
- average of video upload times

For this exercise, there were no additional non-functional requirements related to persistence, scalability and performance e.g. rate of query/updates and number of devices to consider.

Based on the metrics to be collected per device, the natural choice was to use an in-memory store (hash map) to keep track of the devices statistics. Per the requirements, the statistics only required a fixed size structure for each device. 
- The uptime only requires to keep track of the first, last and number of heartbeats received.
- The average upload time the total number of nanoseconds and the number of samples.

As such, the time complexity for all operations is constant O(1), and space complexity linear O(N) based on the number of devices.

The implementation was structured as a classic service, comprised of an api/handlers (controller) to handle the incoming REST requests, service (business logic to store the data and compute the statistics), and data (in-memory storage).

The configuration options were kept to a minimum, controlled with environment variables or command-line options.

The list of devices to monitor is preloaded from a csv file, and remaining static for the duration of the execution (could be a further improvement to make it dynamic, but not called for in the challenge requirements.)


## Implementation Strategy

Since I am new to Go, and this was a relatively simple and self-contained project, I decided to make extensive use of AI coding tools, mostly Claude Code. This was both as an experiment, and also as way to learn the idioms in Go (eg how to structure a standard project, write unit tests, make good use of the available libs, and properly use common patterns). This proved successful.

I first used to the desktop application to present the problem, explain the requirements as I understood them and with my ideas on how to solve the problem and structure the solution at a high level. As such, I chose NOT to copy the instructions provided werbatim. After a few iterations, the AI and I converged on a plan, which resulted on a [prompt](./claude-code-prompt.md) that I could feed to Claude code (CLI).

The analysis of the problem and interaction with Claude desktop took about 30 min. The initial implementation took 5-6 min of "thinking" by the model. There were mistakes, and between the rework done by the model and manually, I spent an overall time of 2 to 2.5 hours to get a working implementation (including the time to become familiar withthe go toolchain).

Note: For authenticity, this document was *NOT* generated with AI :)



## Topics for further discussion / extension

- other stats (eg Last reboot, first startup time, nb reboots, software version, avg video size, etc, etc etc.) can be added.
If explosion of these, consider defining a framework.
○ How would you modify your data model or code to account for more kinds of metrics?

- authentication
- gateway
- deployment
- redis if multiple instances , for scale.
- robustness (hearbeats received out-of-order, or more often than expected. buckets?)
- average since the dawn of time (at least from program  or DB persistence time :) ) is of limited use.
- time series (would be different problem)
- dynamic reloading/confguraiton of devices


You may also use any external libraries that you wish, but you really shouldn’t need to use anything fancy (e.g. databases).



● Include a short write up of your solution, answering the following questions to the best of your ability:
○ Discuss your solution’s runtime complexity.




We strongly suggest first implementing a simple, well-written, correct, and (reasonably) performant program before attempting to optimize it further.
● If you have more time and are confident in your working solution, we encourage you to elaborate on your solution to showcase your architecture and code organization skills. Some ideas:
○ How would you tackle security, testing and deployment?
○ Possibly include a diagram to illustrate how you would structure an alpha prototype.
● If you do expand on your solution after completing the basic requirements, consider using git to commit working checkpoints, in order for us to see your progress and give credit for past versions if the final version does not pan out.

