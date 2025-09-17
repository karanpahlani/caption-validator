# Backend Golang Dev Takehome
The work that the Media Analytics Framework (MAF) Team does is primarily with processing media at scale. This media
can be Video, Captions, Images, or Audio as primary inputs. Each of these media types can have various formats, codecs,
containers, and delivery mechanisms.

Captions are one of the best and most cost-effective ways of providing information- text is much smaller than audio
or video, and is very easy to extract valuable information from.

## Assignment
Design and build a program that can be used to validate captions files. Here are the requirements:
1. The program can take either *WebVTT* or *SRT* files. Other file types would produce an exit code of 1.
2. Validates that the captions cover a user-specified percentage of the total time between a user-specified 
`t_start` and `t_end`.
3. Packages the text of the captions and sends them to a webserver endpoint (user-configurable). Should be sent as
plaintext, and the response will be a JSON object with a single field `lang`, where there are multiple possible result
values, i.e. `en-US`, `en-GB`, `es-ES`, `es-MX`, etc. The only acceptable response value should be `en-US`.
4. The failed validations should be printed to the command line as JSON objects, with a `type` and further description
as to why it failed. Validation failures are NOT failures of the program, and should still exit with a `0` code.
5. The program should be written in GoLang.
6. The program should be executable by building a Dockerfile, included in the response.

### Input
`programname <validation-flags> captions-filepath`

* `programname` - go run, docker run, or the built go executable
* `<validation-flags>` - flags and values required for validation of the captions file
* `captions-filepath` - the path to the captions file

Please note that there are NO flags for specifying the captions format!

### Output
```
{"type": "caption_coverage", ...}
{"type": "incorrect_language", ...}
```
Note that there should be NO output if the captions are valid. Include extra details in other fields that will be
helpful in addressing the validation errors.

Also note that errors in running the validation program are NOT the same as validation errors.

### Keep in Mind
1. Tests are important!
2. Don't be afraid of comments.
3. There are many sample captions files available online, in public GitHub repositories.
4. Some captions file formats are binary. The standard linux program `file` can help you determine this.
5. Logs should go to a different stream than results.
6. All errors should be gracefully handled, we shouldn't see any stack traces.
7. We are looking for good design decisions, so try and think what would be best to make the code as maintainable
and extensible as possible.
8. There is always room for optimization, but this exercise is intended to take 2-3 hours at most. If you have
more ideas for improvements, feel free to document them as TODOs.
9. Use of AI is encouraged! Our team is an AI team, and so the proficient use of it is expected. On the other hand, you
are also responsible for understanding your own code. This means being able to explain each line of code with a 
reasonable level of detail.