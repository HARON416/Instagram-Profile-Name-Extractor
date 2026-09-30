# Instagram-Profile-Name-Extractor

Run `go run .` from the folder containing `profiles.xlsx`. On Windows, you can also use `run-windows.bat`.

After scraping finishes, the program creates a separate `profiles_results-<unique number>.xlsx` in the same folder and prints its full path. The source workbook and earlier results are preserved. Open the new results file to see the names.

Each sheet uses column B for NAME unless its first row contains a USERNAME header, in which case it uses column C. Header matching ignores capitalization and surrounding spaces; username cells may be empty.

If no nonempty names are captured, no results file is created. Check the terminal warnings. A separate output file avoids overwriting the input, but does not resolve scraping failures or lack of permission to write in the folder.
