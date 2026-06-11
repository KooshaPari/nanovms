#ifndef NVMS_SYSCALLS_H
#define NVMS_SYSCALLS_H

#include <stdint.h>

typedef struct nvms_exec_args {
  const char *path;
  const char *const *argv;
  const char *const *envp;
} nvms_exec_args;

int32_t nvms_focus(uint32_t window_id);
void nvms_exit(int32_t code);
int32_t nvms_exec(const nvms_exec_args *args);

#endif
