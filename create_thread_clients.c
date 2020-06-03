#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <sys/wait.h>

void run_client(int client_num) {
	char *client_num_string = malloc(8*sizeof(char) + 10);
	bzero(client_num_string, (8*1+10));

	sprintf(client_num_string, "client_%d", client_num);
	char *args[]={"./executable", "-a", ">", client_num_string, NULL};

	printf("Executing client...\n");
	fflush(stdout);
	int retExec = execvp(args[0], args);

	if (retExec < 0) {
		printf("ERROR: exec failed with status %d and errno %d.\n", retExec, errno);
		fflush(stdout);
		exit(errno);
	}

}

void *wait_for_client(void *args) {
	int *args_int = (int *)args;

	int client_num = args_int[0];
	pid_t fork_pid = args_int[1];

	wait(&fork_pid);

	printf("Client %d crashed! Check logs.\n", client_num);
	fflush(stdout);

	exit(1);

	return NULL;
}

int main(int argc, char const* argv[])
{
	if (argc != 1) {
		printf("wrong number of arguments: %d\nexpected 1, since number of clients is an environment variable\n", argc);
		fflush(stdout);
		return 1;
	}

	char* p;
	long LONG_NUM_CLIENTS = strtol(getenv("NUM_CLIENTS"), &p, 10);
	if (*p != '\0' || errno != 0) {
		return 1;
	}

	int NUM_CLIENTS = LONG_NUM_CLIENTS;
	pthread_t waiting_threads_ids[NUM_CLIENTS];
	int thread_args[NUM_CLIENTS][2];

	printf("Starting %d clients (threads)...\n", NUM_CLIENTS);
	fflush(stdout);

	for(int i = 0; i < NUM_CLIENTS; i++) {
		printf("Creating client %d\n", i);
		fflush(stdout);

		pid_t fork_id = fork();

		if (fork_id == 0) {
			// CHILD
			run_client(i);
		}

		thread_args[NUM_CLIENTS][0] = i;
		thread_args[NUM_CLIENTS][1] = fork_id;		

		pthread_create(&waiting_threads_ids[i], NULL, wait_for_client, thread_args);

		printf("Created client %d\n", i);
		fflush(stdout);

		sleep(1);
	}

	printf("Finished creating clients.\n");

	for(int i = 0; i < NUM_CLIENTS; i++) {
		pthread_join(waiting_threads_ids[i], NULL);
	}

	printf("Thread crashed... Check logs.\n");
	fflush(stdout);

	return 0;
}
