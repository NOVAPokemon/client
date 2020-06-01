#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>

void *run_client(void *func_args) {
	int *client_num = (int *)func_args;

	char *client_num_string = malloc(8*sizeof(char) + sizeof(int));
	sprintf(client_num_string, "client_%d", *client_num);
    char *args[]={"./executable", "-a", ">", client_num_string, NULL};

    printf("Executing client...");

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
		printf("wrong number of arguments: %d\nexpected 1, since number of clients is an environment variable\n", argc);
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

	int client_num;
	for(int i = 0; i < NUM_CLIENTS; i++) {
		printf("Creating client %d\n", i);
   		client_num = i;
		pthread_t thread_id;

		pthread_create(&thread_id, NULL, run_client, (void *)&client_num);
		pthread_ids[i] = thread_id;
		printf("Created client %d\n", i);
		
		sleep(1);
	}

	for(int i = 0; i < NUM_CLIENTS; i++) {
		pthread_join(pthread_ids[i], NULL);
	}

	return 0;
}
