# Generate a full sliding window visualization with all steps
fig, ax = plt.subplots(figsize=(12, 6))

for step, t in enumerate(trace):
    ax.clear()
    ax.set_xlim(-1, text_len)
    ax.set_ylim(0, 1)
    
    # Plot the text as individual characters
    for i, char in enumerate(text):
        ax.text(i, 0.6, char, ha='center', va='center', fontsize=12, bbox=dict(facecolor='lightgray', edgecolor='black', boxstyle='round,pad=0.3'))

    # Determine the current comparison index
    if "Comparing" in t:
        comparing_idx = int(t.split('[')[1].split(']')[0])
        pattern_idx = int(t.split('pattern[')[1].split(']')[0])

        # Highlight the character being compared in yellow
        ax.text(comparing_idx, 0.6, text[comparing_idx], ha='center', va='center', fontsize=12, bbox=dict(facecolor='yellow', edgecolor='black', boxstyle='round,pad=0.3'))

        # Plot the pattern at its current position (sliding window)
        start_pos = comparing_idx - pattern_idx
        for j, char in enumerate(pattern):
            if start_pos + j >= 0 and start_pos + j < text_len:
                ax.text(start_pos + j, 0.3, char, ha='center', va='center', fontsize=12, bbox=dict(facecolor='lightblue', edgecolor='black', boxstyle='round,pad=0.3'))

    ax.set_xticks(range(text_len))
    ax.set_xticklabels(list(text))
    ax.set_yticks([])
    ax.set_title(f"Step {step + 1}: {t}", fontsize=14)
    
    plt.pause(1)  # Pause to show each step
    plt.draw()

plt.show()
