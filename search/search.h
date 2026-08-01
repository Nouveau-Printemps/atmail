#pragma once
#include <stdlib.h>

typedef struct {
    bool found;
    size_t pos;
    size_t rest;
} SearchResponse;

// Search takes two null-terminated string.
SearchResponse search(char *, char *);
