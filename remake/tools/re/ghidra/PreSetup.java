import ghidra.app.script.GhidraScript;
import ghidra.program.model.address.*;
import java.nio.file.*;

public class PreSetup extends GhidraScript {
    @Override
    public void run() throws Exception {
        String[] args = getScriptArgs();
        setAnalysisOption(currentProgram, "Decompiler Parameter ID", "true");
        SegmentedAddress sm = (SegmentedAddress) currentProgram.getMinAddress();
        int baseSeg = sm.getSegment();
        for (String line : Files.readAllLines(Paths.get(args[0]))) {
            line = line.trim();
            if (line.isEmpty()) continue;
            String[] p = line.split(":");
            int seg = Integer.parseInt(p[0], 16) + baseSeg;
            int off = Integer.parseInt(p[1], 16);
            Address a = toAddr(String.format("%04x:%04x", seg, off));
            disassemble(a);
            createFunction(a, String.format("f_%04x_%04x", seg - baseSeg, off));
        }
    }
}
