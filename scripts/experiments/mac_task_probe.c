// Diagnostic only: requests a task port, reports permission, then releases it.
// Does not read/write memory, attach a debugger, inject or change system policy.
#include <mach/mach.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <limits.h>
int main(int argc, char **argv) {
    if (argc != 2) { fprintf(stderr, "usage: mac-task-probe WOW_PID\n"); return 2; }
    char *end; errno = 0;
    long n = strtol(argv[1], &end, 10);
    if (errno || *end || n <= 0 || n > INT_MAX) return 2;
    mach_port_t task = MACH_PORT_NULL;
    kern_return_t result = task_for_pid(mach_task_self(), (pid_t)n, &task);
    printf("task_for_pid: %d (%s)\n", result, mach_error_string(result));
    if (task != MACH_PORT_NULL) mach_port_deallocate(mach_task_self(), task);
    return result == KERN_SUCCESS ? 0 : 1;
}
