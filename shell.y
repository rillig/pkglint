%{
package main
%}

%union {
	Word *MkShWord
}

%token tkWORD
%token tkASSIGNMENT_WORD
%token tkNAME
%token tkNEWLINE
%token tkIO_NUMBER
%token tkBACKGROUND
%token tkPIPE tkSEMI
%token tkAND tkOR tkSEMISEMI
%token tkLT tkGT tkLTLT tkGTGT tkLTAND tkGTAND  tkLTGT tkLTLTDASH tkGTPIPE
%token tkIF tkTHEN tkELSE tkELIF tkFI tkDO tkDONE
%token tkCASE tkESAC tkWHILE tkUNTIL tkFOR
%token tkLPAREN tkRPAREN tkLBRACE tkRBRACE tkEXCLAM
%token tkIN

%%

program          : list separator
                 | list
                 ;
list             : list separator_op and_or
                 |                   and_or
                 ;
and_or           :                        pipeline
                 | and_or tkAND linebreak pipeline
                 | and_or tkOR  linebreak pipeline
                 ;
pipeline         :          pipe_sequence
                 | tkEXCLAM pipe_sequence
                 ;
pipe_sequence    :                                command
                 | pipe_sequence tkPIPE linebreak command
                 ;
command          : simple_command
                 | compound_command
                 | compound_command redirect_list
                 | function_defn
                 | function_defn redirect_list
                 ;
compound_command : brace_group
                 | subshell
                 | for_clause
                 | case_clause
                 | if_clause
                 | while_clause
                 | until_clause
                 ;
subshell         : tkLPAREN compound_list tkRPAREN
                 ;
compound_list    : linebreak term
                 | linebreak term separator
                 ;
term             : term separator and_or
                 |                and_or
                 ;
for_clause       : tkFOR name linebreak                            do_group
                 | tkFOR name linebreak in          sequential_sep do_group
                 | tkFOR name linebreak in wordlist sequential_sep do_group
                 ;
name             : tkNAME                     /* Apply rule 5 */
                 ;
in               : tkIN                       /* Apply rule 6 */
                 ;
wordlist         : wordlist tkWORD
                 |          tkWORD
                 ;
case_clause      : tkCASE tkWORD linebreak in linebreak case_list    tkESAC
                 | tkCASE tkWORD linebreak in linebreak case_list_ns tkESAC
                 | tkCASE tkWORD linebreak in linebreak              tkESAC
                 ;
case_list_ns     : case_list case_item_ns
                 |           case_item_ns
                 ;
case_list        : case_list case_item
                 |           case_item
                 ;
case_selector    : tkLPAREN pattern tkRPAREN
                 |          pattern tkRPAREN
                 ;
case_item_ns     : case_selector linebreak
                 | case_selector linebreak term linebreak
                 | case_selector linebreak term separator_op linebreak
                 ;
case_item        : case_selector linebreak     tkSEMISEMI linebreak
                 | case_selector compound_list tkSEMISEMI linebreak
                 ;
pattern          :                tkWORD         /* Apply rule 4 */
                 | pattern tkPIPE tkWORD         /* Do not apply rule 4 */
                 ;
if_clause        : tkIF compound_list tkTHEN compound_list else_part tkFI
                 | tkIF compound_list tkTHEN compound_list           tkFI
                 ;
else_part        : tkELIF compound_list tkTHEN compound_list
                 | tkELIF compound_list tkTHEN compound_list else_part
                 | tkELSE compound_list
                 ;
while_clause     : tkWHILE compound_list do_group
                 ;
until_clause     : tkUNTIL compound_list do_group
                 ;
function_defn    : fname tkLPAREN tkRPAREN linebreak compound_command   /* Apply rule 9 */
                 ;
fname            : tkNAME                            /* Apply rule 8 */
                 ;
brace_group      : tkLBRACE compound_list tkRBRACE
                 ;
do_group         : tkDO compound_list tkDONE           /* Apply rule 6 */
                 ;
simple_command   : cmd_prefix cmd_word cmd_suffix
                 | cmd_prefix cmd_word
                 | cmd_prefix
                 | cmd_name cmd_suffix
                 | cmd_name
                 ;
cmd_name         : tkWORD                   /* Apply rule 7a */
                 ;
cmd_word         : tkWORD                   /* Apply rule 7b */
                 ;
cmd_prefix       :            io_redirect
                 |            tkASSIGNMENT_WORD
                 | cmd_prefix io_redirect
                 | cmd_prefix tkASSIGNMENT_WORD
                 ;
cmd_suffix       :            io_redirect
                 |            tkWORD
                 | cmd_suffix io_redirect
                 | cmd_suffix tkWORD
                 ;
redirect_list    :               io_redirect
                 | redirect_list io_redirect
                 ;
io_redirect      :             io_file
                 | tkIO_NUMBER io_file
                 |             io_here
                 | tkIO_NUMBER io_here
                 ;
io_file          : tkLT     filename
                 | tkLTAND  filename
                 | tkGT     filename
                 | tkGTAND  filename
                 | tkGTGT   filename
                 | tkLTGT   filename
                 | tkGTPIPE filename
                 ;
filename         : tkWORD                      /* Apply rule 2 */
                 ;
io_here          : tkLTLT     here_end
                 | tkLTLTDASH here_end
                 ;
here_end         : tkWORD                      /* Apply rule 3 */
                 ;
newline_list     :              tkNEWLINE
                 | newline_list tkNEWLINE
                 ;
linebreak        : newline_list
                 | /* empty */
                 ;
separator_op     : tkBACKGROUND
                 | tkSEMI
                 ;
separator        : separator_op linebreak
                 | newline_list
                 ;
sequential_sep   : tkSEMI linebreak
                 | tkNEWLINE linebreak
                 ;
