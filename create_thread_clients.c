#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include<unistd.h>

void *run_client(void *vargp) {
    char *args[]={"./$executable", "-a", NULL};
    int retExec = execvp(args[0], args);

	if (retExec < 0) {
	    printf("ERROR: exec failed with status %d and errno %d.\n", retExec, errno);
	    exit(errno);
	}

	return NULL;
}

int main(int argc, char const* argv[])
{
	if (argc != 1) {
		printf("wrong number of arguments: %d\nexpected 1, since number of clients is an environment variable", argc);
		return 1;
	}

	char* p;
	long LONG_NUM_CLIENTS = strtol(getenv("NUM_CLIENTS"), &p, 10);
	if (*p != '\0' || errno != 0) {
        return 1;
    }

    int NUM_CLIENTS = LONG_NUM_CLIENTS;
    pthread_t pthread_ids[NUM_CLIENTS];

    printf("Starting %d clients (threads)...", NUM_CLIENTS);

	for(int i = 0; i < NUM_CLIENTS; i++) {
		pthread_t thread_id;
		pthread_create(&thread_id, NULL, run_client, NULL);
		pthread_ids[i] = thread_id;
	}

	for(int i = 0; i < NUM_CLIENTS; i++) {
		pthread_join(pthread_ids[i], NULL);
	}

	return 0;
}
